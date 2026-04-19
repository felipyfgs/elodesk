## SMS Channel

### Overview

The SMS channel supports three providers: **Twilio**, **Bandwidth**, and **Zenvia**. All providers share a common `Channel::Sms` type with provider-specific credentials and webhook handling.

### Provisioning

#### Twilio

1. Create a Twilio account and get a phone number from the [Twilio Console](https://console.twilio.com/).
2. Note your `Account SID` and `Auth Token`.
3. Provision via API:
   ```bash
   POST /api/v1/accounts/:aid/inboxes/sms
   {
     "name": "My Twilio SMS",
     "provider": "twilio",
     "phoneNumber": "+14155551234",
     "providerConfig": {
       "twilio": {
         "accountSid": "AC...",
         "authToken": "...",
         "messagingServiceSid": "MG... (optional)"
       }
     }
   }
   ```
4. Configure webhooks in Twilio Console → Phone Number → Messaging:
   - **A message comes in**: `https://your-domain/webhooks/sms/twilio/:identifier`
   - **Status callback**: `https://your-domain/webhooks/sms/twilio/:identifier/status`

#### Bandwidth

1. Create a Bandwidth account and get credentials from the [Dashboard](https://dashboard.bandwidth.com/).
2. Create an Application and note `Account ID`, `Application ID`, `Basic Auth User`, `Basic Auth Pass`.
3. Provision via API with `provider: "bandwidth"` and `providerConfig.bandwidth`.
4. Configure callbacks in Bandwidth Dashboard → Application → Callbacks.

#### Zenvia

1. Create a Zenvia account and get an API token from the [Portal](https://app.zenvia.com/).
2. Provision via API with `provider: "zenvia"` and `providerConfig.zenvia`.
3. Configure webhooks in Zenvia Portal.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DEFAULT_PHONE_REGION` | Default region for phone normalization | `BR` |

### Webhook URLs

After provisioning, the API returns:
```json
{
  "webhookUrls": {
    "primary": "https://your-domain/webhooks/sms/twilio/abc123",
    "status": "https://your-domain/webhooks/sms/twilio/abc123/status"
  }
}
```

### Signature Verification

- **Twilio**: `X-Twilio-Signature` header (HMAC-SHA1)
- **Bandwidth**: HTTP Basic Auth
- **Zenvia**: `X-Zenvia-Signature` header (HMAC-SHA256)

---

## Telegram Channel (`Channel::Telegram`)

### Overview

Receives messages from a Telegram Bot via the Telegram Bot API. No global ENV vars required — bot tokens are per-channel and encrypted at rest.

### Creating a bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the bot token (format: `123456:ABC-DEF...`)

### Provisioning

```bash
POST /api/v1/accounts/:aid/inboxes/telegram
{
  "name": "My Telegram Bot",
  "botToken": "<bot-token-from-botfather>"
}
```

The endpoint automatically:
1. Calls `getMe` to validate the token and fetch `bot_name`
2. Generates a `webhook_identifier` (opaque token) and `secret_token` (32 bytes)
3. Registers the webhook with Telegram via `setWebhook`
4. Creates the inbox with `channel_type = "Channel::Telegram"`

### Webhook

Telegram delivers updates to `https://your-host/webhooks/telegram/<webhook_identifier>`, validated via the `X-Telegram-Bot-Api-Secret-Token` header.

### Supported inbound types

Text, photo, video, audio, voice, document, sticker, animation, location, contact, video_note. Edited messages are logged but ignored (MVP). Callback queries from inline keyboards are processed as messages.

**Groups/supergroups are silently ignored** (MVP is 1:1 only).

### Outbound

Markdown is converted to Telegram HTML subset (`<b>`, `<i>`, `<u>`, `<s>`, `<code>`, `<pre>`, `<a>`). Reply threading (`reply_to_message_id`) and inline keyboards (`reply_markup`) are supported via `content_attributes`.

### Media download

Media is downloaded lazily from Telegram CDN on first view, then cached in MinIO.

### Deleting a channel

```
DELETE /api/v1/accounts/:aid/inboxes/:id/telegram
```

Calls `deleteWebhook` on Telegram before removing the local record.

---

## Web Widget Channel (`Channel::WebWidget`)

### Overview

Embeddable chat widget for customer websites. Visitors start as anonymous contacts and can be identified via HMAC verification. Real-time messaging via SSE with polling fallback.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `WIDGET_PUBLIC_BASE_URL` | Public URL for the widget bundle | `http://localhost:3001` |
| `WIDGET_JWT_SECRET` | Secret key for signing visitor JWTs | (required) |
| `WIDGET_SESSION_TTL_DAYS` | Visitor session TTL in days | `30` |

### Provisioning

```bash
POST /api/v1/accounts/:aid/inboxes/web_widget
{
  "name": "My Website Chat",
  "websiteUrl": "https://mysite.com",
  "widgetColor": "#0084FF",
  "welcomeTitle": "Hello!",
  "welcomeTagline": "How can we help?",
  "replyTime": "in_a_few_minutes"
}
```

Response includes `websiteToken`, `embedScript` (ready to paste), and `hmacToken` (shown once).

### Embed Script

Copy the `embedScript` from the provisioning response and paste it before `</body>` on your site:

```html
<script src="https://widget.elodesk.io/widget/<websiteToken>" data-website-token="<websiteToken>" defer></script>
```

### Identify with HMAC

To verify visitor identity, your backend computes an HMAC and passes it to the widget:

**Node.js:**
```js
const crypto = require('crypto');
const hash = crypto.createHmac('sha256', hmacToken).update('user@acme.com').digest('hex');
```

**Ruby:**
```ruby
require 'openssl'
hash = OpenSSL::HMAC.hexdigest('SHA256', hmac_token, 'user@acme.com')
```

**PHP:**
```php
$hash = hash_hmac('sha256', 'user@acme.com', $hmacToken);
```

Then call `POST /api/v1/widget/identify` with `{identifier, identifierHash}`.

### HMAC Token Rotation

```bash
POST /api/v1/accounts/:aid/inboxes/:id/rotate_hmac
```

Returns the new `hmacToken` once. Update your backend integration immediately.

### CDN Setup

Build the widget bundle:
```bash
cd widget && npm install && npm run build
```

Upload `dist/widget.js` to your CDN (S3/CloudFront, Cloudflare R2). Set `WIDGET_PUBLIC_BASE_URL` to the CDN URL.
