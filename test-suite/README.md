# Test Suite

## Quickstart

### 1. Start the Example Server

Make sure you are in the project root, then run:

```bash
go run servers/cmd.go
````

---

### 2. Run the Test Suite

From the project root, run:

```bash
go test -race -cover -covermode=atomic ./...
```

Notes:

* `-race` enables **race condition detection**
* `-cover` generates a **coverage report**
* `-covermode=atomic` is recommended for concurrent tests

The tests cover both **HTTP long-polling** and **WebSocket** transports (see `test-suite_test.go`).

---

## Requirements

* Go 1.24+

---

## Troubleshooting

* **Tests failing**
  Make sure the local server is running and listening on the expected port.
  You can verify with:

```bash
curl http://localhost:3000
```
