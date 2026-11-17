# Fitness Tracker

A personal, single-user fitness tracker: one Go binary that serves a web UI (HTML + HTMX),
backed by MongoDB Atlas. There is no authentication of its own;
it is meant to be reached only over a private network such as Tailscale.

## Layout

```
main.go                 starts the web UI
internal/
  config/ database/ models/ service/
                        configuration, MongoDB access, data types and workout logic
  web/                  UI page handlers (home, profile, exercises, routines, recorder, analytics)
  ui/                   HTML templates and static assets, embedded in the binary
  api/                  the UI's data source: types, the Backend interface, and its database implementation
  live/                 local live-session file for the workout in progress
```

The UI is served at `/`. Everything it needs, including its scripts, styles and icon font,
is embedded in the binary, so it works without internet access.

## Running

1. Copy `.env.example` to `.env` and fill in `MONGODB_URI` (the database is created and
   seeded with the built-in exercise list on first start).
2. `go run .` (or `go build` and run the binary), then open `http://localhost:8090`.

Settings are in `.env.example`: `PORT`, `SESSION_FILE`, and a few optional ones.
Requires Go 1.24+.

## Live workout session

While a workout is running, the app keeps it in `SESSION_FILE` and pushes every change to
the database session. Pages read from the file, so recording a set doesn't wait on the
database. If the database is unreachable, changes stay in the file and sync on the next
action; Finish waits until they have. If the stored session is missing (for example it was
deleted by hand) it is recreated from the routine and the local work is kept.

## Tests

`go test ./...` runs without a database. It covers the live session, routing, the
comparison and history logic and the exercise seed list.

## Icon font

`internal/ui/static/material-symbols-rounded.woff2` is a subset of Material Symbols Rounded
containing only the icons the UI uses. When you add a new icon, regenerate it by requesting
`https://fonts.googleapis.com/css2?family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@24,400,0,0&icon_names=<all icon names, comma separated>`
with a Chrome user agent and saving the `.woff2` it points to.

## License

CC BY-NC. See [LICENSE](LICENSE).
