# Basic CRUD Application

A real-time CRUD (Create, Read, Update, Delete) application using Socket.IO in Go.

## Features

- Create, read, update, and delete TODO items
- All changes are broadcast to all connected clients in real-time
- Acknowledgement (ack) support for confirmation of operations
- Thread-safe in-memory storage

## How to run

```bash
go run main.go
```

The server starts on `http://localhost:3000` by default. Set the `PORT` environment variable to use a different port.

## Events

### Client → Server

| Event | Payload | Ack | Description |
|-------|---------|-----|-------------|
| `todo:create` | `{ title }` | `TodoItem` | Create a new item |
| `todo:read` | — | `[]TodoItem` | List all items |
| `todo:update` | `{ id, title, completed }` | `TodoItem` | Update an existing item |
| `todo:delete` | `{ id }` | `{ id }` | Delete an item |

### Server → Client

| Event | Payload | Description |
|-------|---------|-------------|
| `todo:list` | `[]TodoItem` | Full item list sent on connection |
| `todo:created` | `TodoItem` | A new item was created |
| `todo:updated` | `TodoItem` | An item was updated |
| `todo:deleted` | `{ id }` | An item was deleted |

### TodoItem

```json
{
  "id": 1,
  "title": "Buy groceries",
  "completed": false
}
```

## Running tests

```bash
go test -v -race ./...
```
