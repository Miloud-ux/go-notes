# go-notes

A simple REST API for notes built with Go's standard `net/http` package and SQLite.

## Features

- Create, list, get, and delete notes
- SQLite storage (in-memory store available for tests)
- Middleware: request ID, logging, panic recovery
- Graceful shutdown

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
