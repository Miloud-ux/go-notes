# go-notes

A simple REST API for notes built with Go's standard `net/http` package and SQLite, with a reverse proxy that load balances across backends.

## Features

- Create, list, get, and delete notes
- SQLite storage (in-memory store available for tests)
- Middleware: request ID, logging, panic recovery
- Graceful shutdown
- Reverse proxy with round robin load balancing

## Endpoints

| Method | Path         | Description       |
|--------|--------------|-------------------|
| GET    | /health      | Health check      |
| POST   | /notes       | Create a note     |
| GET    | /notes       | List all notes    |
| GET    | /notes/{id}  | Get a note        |
| DELETE | /notes/{id}  | Delete a note     |

## Run

```bash
go run .
```

The server listens on `:8000`.

Run more backends by starting the API on other ports, for example `:8001`.

## Reverse Proxy

The proxy in `proxy/` forwards requests to the backends using round robin.

```bash
go run ./proxy
```

It listens on `:8080` and cycles requests between `http://localhost:8000` and `http://localhost:8001` (configured in `proxy/main.go`). This is a simplified demo and is not production ready.

## Test

```bash
go test
```

## Example

```bash
curl -X POST localhost:8000/notes -d '{"text":"hello"}'
curl localhost:8000/notes/1
curl localhost:8000/notes
curl -X DELETE localhost:8000/notes/1
```
