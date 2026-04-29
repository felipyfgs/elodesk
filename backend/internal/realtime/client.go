package realtime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/websocket"

	"backend/internal/logger"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 4096

	membershipLookupTimeout = 5 * time.Second
)

type clientMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// pongMessage is the canned reply to a client `ping` text frame. Pre-marshalled
// to avoid per-ping allocation; the payload is intentionally empty since the
// client only needs to see *any* response to keep its heartbeat alive.
var pongMessage = []byte(`{"type":"pong"}`)

type joinPayload struct {
	ID int64 `json:"id"`
}

func NewClient(hub *Hub, conn *websocket.Conn, userID int64, checker MembershipChecker) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		userID:  userID,
		rooms:   make(map[string]struct{}),
		send:    make(chan []byte, 256),
		checker: checker,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Warn().Str("component", "realtime").Int64("userId", c.userID).Err(err).Msg("websocket read error")
			}
			break
		}

		var cm clientMessage
		if err := json.Unmarshal(msg, &cm); err != nil {
			logger.Debug().Str("component", "realtime").Int64("userId", c.userID).Err(err).Msg("invalid client message")
			continue
		}

		c.handleMessage(cm)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn().Str("component", "realtime").Int64("userId", c.userID).Err(err).Msg("websocket write error")
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage routes inbound client commands. All join.* variants validate
// membership via the checker before admitting the client to the room: a user
// can only join rooms belonging to an account they are a member of.
// Cross-tenant join attempts are dropped silently (do not leak room existence).
//
// `leave.*` are membership-free: a client can always remove itself from a room
// it is currently in. Unknown/foreign rooms are no-ops.
//
// `ping` is the application-level keepalive sent by clients (vueuse useWebSocket
// heartbeat). The hub answers with `pong` so the heartbeat watchdog sees a
// response on idle conversations — without this, idle sockets die every 30 s.
func (c *Client) handleMessage(cm clientMessage) {
	switch cm.Type {
	case "join.account":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), membershipLookupTimeout)
		defer cancel()
		if !c.checker.UserInAccount(ctx, c.userID, p.ID) {
			logger.Warn().Str("component", "realtime").Int64("userId", c.userID).Int64("accountId", p.ID).Msg("denied join.account")
			return
		}
		c.hub.JoinRoom(c, AccountRoom(p.ID))
		c.hub.JoinRoom(c, UserRoom(p.ID, c.userID))

	case "join.inbox":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), membershipLookupTimeout)
		defer cancel()
		accountID, ok := c.checker.InboxAccount(ctx, p.ID)
		if !ok || !c.checker.UserInAccount(ctx, c.userID, accountID) {
			logger.Warn().Str("component", "realtime").Int64("userId", c.userID).Int64("inboxId", p.ID).Msg("denied join.inbox")
			return
		}
		c.hub.JoinRoom(c, InboxRoom(p.ID))

	case "join.conversation":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), membershipLookupTimeout)
		defer cancel()
		accountID, ok := c.checker.ConversationAccount(ctx, p.ID)
		if !ok || !c.checker.UserInAccount(ctx, c.userID, accountID) {
			logger.Warn().Str("component", "realtime").Int64("userId", c.userID).Int64("conversationId", p.ID).Msg("denied join.conversation")
			return
		}
		c.hub.JoinRoom(c, ConversationRoom(p.ID))

	case "leave.account":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		c.hub.LeaveRoom(c, AccountRoom(p.ID))
		c.hub.LeaveRoom(c, UserRoom(p.ID, c.userID))

	case "leave.inbox":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		c.hub.LeaveRoom(c, InboxRoom(p.ID))

	case "leave.conversation":
		var p joinPayload
		if err := json.Unmarshal(cm.Payload, &p); err != nil {
			return
		}
		c.hub.LeaveRoom(c, ConversationRoom(p.ID))

	case "ping":
		c.enqueue(pongMessage)

	case "event":
		return

	default:
		logger.Debug().Str("component", "realtime").Int64("userId", c.userID).Str("type", cm.Type).Msg("unknown message type")
	}
}

// enqueue is a non-blocking send to the writer pump. If the buffer is full the
// client is too slow to keep up — let the hub drop it on the next broadcast
// (same backpressure path) instead of stalling the read loop here.
func (c *Client) enqueue(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}
