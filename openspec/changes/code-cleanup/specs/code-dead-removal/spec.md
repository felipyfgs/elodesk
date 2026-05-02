## ADDED Requirements

### Requirement: Worker MUST NOT create unused realtime Hub
The asynq worker process SHALL NOT instantiate or run a `realtime.Hub`, as the worker has no HTTP server and does not broadcast realtime events. Any future realtime broadcasting from the worker SHALL use Redis pub/sub or the backend HTTP server's existing hub, not a separate hub instance.

#### Scenario: Worker starts without realtime Hub
- **WHEN** the worker binary starts (`cmd/worker/main.go`)
- **THEN** no `realtime.NewHub()` call is made
- **AND** no `go hub.Run()` goroutine is launched
- **AND** the `backend/internal/realtime` import is not present

#### Scenario: Worker compiles successfully after removal
- **WHEN** `go build ./cmd/worker/` is executed
- **THEN** compilation succeeds with zero errors

### Requirement: TikTok stub SHALL be documented as TODO
The `_ = referencedMessageID` line in `channel/tiktok/send.go` SHALL be replaced with a clear TODO comment indicating the feature is pending implementation, while still discarding the unused parameter to avoid compilation errors.

#### Scenario: TikTok send compiles without unused variable error
- **WHEN** `go build ./internal/channel/tiktok/` is executed
- **THEN** compilation succeeds
- **AND** the `referencedMessageID` parameter produces no unused variable warning
