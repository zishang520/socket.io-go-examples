# Middleware Authentication Example

Demonstrates server-side middleware for token-based authentication with Socket.IO in Go.

## Features

- Namespace-level middleware for connection authentication
- Token validation before connections are established
- Admin-only namespace with additional authorization
- Profile retrieval via acknowledgements
- User connection/disconnection notifications

## How to run

```bash
go run main.go
```

The server starts on `http://localhost:3000` by default. Set the `PORT` environment variable to use a different port.

## Authentication Flow

1. Client connects with auth data: `{ "token": "token-alice" }`
2. Server middleware validates the token
3. If valid, connection proceeds and client receives a `welcome` event
4. If invalid, client receives a `connect_error` event

## Valid Tokens

| Token | Username | Admin |
|-------|----------|-------|
| `token-alice` | Alice | No |
| `token-bob` | Bob | No |
| `token-admin` | Admin | Yes |

## Namespaces

- `/` — Main namespace, requires valid token
- `/admin` — Admin namespace, requires admin token

## Events

### Main Namespace (`/`)

| Event | Direction | Payload | Description |
|-------|-----------|---------|-------------|
| `welcome` | Server → Client | `{ message, username }` | Sent after successful auth |
| `user:connected` | Server → Client | `{ username }` | Broadcast when a user connects |
| `user:disconnected` | Server → Client | `{ username }` | Broadcast when a user disconnects |
| `profile` | Client → Server | — (ack) | Request user profile data |

### Admin Namespace (`/admin`)

| Event | Direction | Payload | Description |
|-------|-----------|---------|-------------|
| `admin:welcome` | Server → Client | `{ message }` | Sent after successful admin auth |
| `admin:action` | Client → Server | `string` | Perform an admin action |
| `admin:action:result` | Server → Client | `{ success, action }` | Result of admin action |

## Running tests

```bash
go test -v -race ./...
```
