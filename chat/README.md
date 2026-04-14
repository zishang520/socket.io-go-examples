# Socket.IO Chat Example

A Go implementation of the classic Socket.IO chat room demo.

## Features

- Multiple users can join a chat room by providing a unique username
- Users can send chat messages to all connected users
- Typing indicator notifications are broadcast to other users
- Join/leave notifications with active user count

## How to run

```bash
go run main.go
```

The server starts on `http://localhost:3000` by default. Set the `PORT` environment variable to use a different port.

## Events

### Client → Server

| Event | Payload | Description |
|-------|---------|-------------|
| `add user` | `string` (username) | Register a username for the connection |
| `new message` | `string` (message) | Send a chat message to the room |
| `typing` | — | Notify others that the user is typing |
| `stop typing` | — | Notify others that the user stopped typing |

### Server → Client

| Event | Payload | Description |
|-------|---------|-------------|
| `login` | `{ numUsers }` | Acknowledgement after successful `add user` |
| `new message` | `{ username, message }` | A chat message from another user |
| `user joined` | `{ username, numUsers }` | A new user joined the room |
| `user left` | `{ username, numUsers }` | A user left the room |
| `typing` | `{ username }` | Another user is typing |
| `stop typing` | `{ username }` | Another user stopped typing |

## Running tests

```bash
go test -v -race ./...
```
