# Fitness Tracker

A personal, single-user fitness tracker: one Go binary that serves a web UI (HTML + HTMX)
and the JSON API behind it, backed by MongoDB Atlas. There is no authentication of its own;
it is meant to be reached only over a private network such as Tailscale.

## Layout

```
main.go                 starts the API and the UI on one port
internal/
  config/ database/ handlers/ middleware/ models/ routes/ service/ utils/
                        the backend: JSON API served under /api
  web/                  UI page handlers (home, profile, exercises, routines, recorder, analytics)
  ui/                   HTML templates and static assets, embedded in the binary
  api/                  the UI's client for the JSON API (over loopback)
  live/                 local live-session file for the workout in progress
```

The UI is served at `/`, the API at `/api/...` (for example `/api/me`).

## Running

1. Copy `.env.example` to `.env` and fill in `MONGODB_URI` (the database is created and
   seeded with the built-in exercise list on first start).
2. `go run .` (or `go build` and run the binary), then open `http://localhost:8090`.

Settings are in `.env.example`: `PORT`, `BIND_ADDR`, `SESSION_FILE`, and a few optional ones.
Requires Go 1.24+.

## Live workout session

While a workout is running, the app keeps it in `SESSION_FILE` and pushes every change to
the backend session. Pages read from the file, so recording a set doesn't wait on the
database. If the backend is unreachable, changes stay in the file and sync on the next
action; Finish waits until they have. If the backend session has expired (sessions expire
after 48 hours without updates) it is recreated from the routine and the local work is kept.

## Tests

`go test ./...` runs without a database. It covers the live session, routing, the
comparison logic and the exercise seed list.

## License

CC BY-NC. See [LICENSE](LICENSE).
