# socket.io-go-examples

Example repository demonstrating usage and conformance tests for the Go Socket.IO implementation by zishang520. This repo contains multiple example applications and an automated test-suite that exercises Engine.IO and Socket.IO behaviors.

## Examples

| Example | Description |
|---------|-------------|
| [benchmark](./benchmark/) | Memory/goroutine leak benchmark with high-frequency connect/disconnect |
| [chat](./chat/) | Classic chat room with usernames, typing indicators, and join/leave notifications |
| [basic-crud-application](./basic-crud-application/) | Real-time CRUD operations on a shared TODO list with broadcast updates |
| [middleware-auth](./middleware-auth/) | Token-based authentication middleware with admin namespace authorization |
| [test-suite](./test-suite/) | Protocol conformance tests for Engine.IO and Socket.IO |

## Quick Start

Each example is a standalone Go module. To run any example:

```bash
cd examples/<example-name>
go run main.go
```

To run tests:

```bash
cd examples/<example-name>
go test -v -race ./...
```

## Example Features

### Chat
- Multiple users join with unique usernames
- Real-time message broadcasting
- Typing indicator notifications
- User join/leave events with active user count

### Basic CRUD Application
- Create, read, update, delete TODO items
- All changes broadcast to connected clients in real-time
- Acknowledgement (ack) support for operation confirmation
- Thread-safe in-memory storage

### Middleware Auth
- Namespace-level middleware for connection authentication
- Token validation before connection is established
- Admin-only namespace with additional authorization
- Profile retrieval via acknowledgements

### Test Suite
- Engine.IO handshake (HTTP long-polling + WebSocket)
- Engine.IO heartbeat (ping/pong + timeout)
- Engine.IO session close and upgrade
- Socket.IO connect/disconnect with namespaces
- Socket.IO message passing (plain-text, binary, ack)
- Payload limits, edge cases, and session management