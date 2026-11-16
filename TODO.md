# TODO

What is left after merging the backend into this workspace. Grouped by area; the first
group is what stands between this repo and retiring the old apps.

## Before the old apps can be retired

- [ ] **Run the merged server against a database and click through it.** Nothing here has
  touched MongoDB yet: Atlas rejected this machine's TLS handshake (most likely its IP is
  not in Atlas Network Access). Use a separate database name, e.g.
  `MONGODB_DBNAME=fitness_app_test`, never `fitness_app`.
- [ ] **Dockerfile and `docker-compose.yaml`** for the single binary: multi-stage build,
  `CGO_ENABLED=0`, `linux/arm64` for the Orange Pi, a volume for `SESSION_FILE`, env from
  `.env`. Untested so far (no Docker on the dev machine).
  - [ ] Decide how it is deployed on the Orange Pi: Docker, or the plain binary under systemd.
  - [ ] Decide `BIND_ADDR` (Tailscale IP) for the new deployment.
- [ ] **Confirm feature parity with Streamlit**, page by page, using real data.
- [ ] **Remove the Streamlit app** (`fitness-frontend`) once parity is confirmed. Until
  then it keeps working if started with `BACKEND_URL=http://opi5:8090/api` (the API moved
  under `/api`). Do not delete it without an explicit go-ahead.
- [ ] **Stop the old services on the Orange Pi** (Streamlit on 8501, the old backend on 8090)
  when the new binary takes over port 8090.
- [ ] **Rotate the Atlas database password.** It was pasted into a chat while testing.
- [ ] **Try a real workout end to end** with the live-session file on real data, including
  killing the server mid-workout and restarting it.

## Architecture

- [ ] **Replace the loopback HTTP hop with direct calls** to the `database`/`service`
  packages (put an interface between `internal/web` and the data source). Then drop
  `internal/api`'s HTTP client, and have the live session push straight to the database.
  - [ ] Decide what happens to `MOCK_ANALYTICS` (keep as a second implementation of the
    same interface, or drop it once real data exists).
- [ ] Decide the final module path (currently `fitness-tracker`, the backend's) and the
  repo name / remote. There is no git remote yet.
- [ ] Decide whether the API should stay exposed on the same listener as the UI. It has no
  authentication of its own and relies on the private network (Tailscale).

## Backend review items not fixed yet

- [ ] `PATCH /session/update` drops each exercise's `name`, so `GET /session/data` does one
  database lookup per exercise to put it back. Store the name, or stop returning it.
- [ ] Sessions still expire after 48 hours (TTL index on `lastUpdated`). The UI recovers
  from that now, but decide whether 48h is the right value.
- [ ] `UpdateUser` never checks whether anything matched (the other updates now return 404).
- [ ] Some cursors are never closed (`GetUserWorkouts`, `GetExerciseList`), and
  `GetExerciseList` ignores cursor errors.
- [ ] `GET /routines/list` returns JSON `null` instead of `[]` when there are none.
- [ ] `/me` creates a default user named "Athlete" when none exists; decide whether that is
  still wanted for a fresh database.
- [ ] No tests cover the `database` package or the handlers that need MongoDB. Needs a test
  database or a fake behind an interface (see Architecture).

## UI

- [ ] **Vendor the remaining CDN assets.** HTMX (unpkg) and the Material Symbols font
  (Google Fonts) are still loaded from the internet, so the UI needs a connection to render.
  Chart.js and SortableJS are already embedded.
- [ ] **Visual and spacing pass** on the pages not reworked yet: Home, Profile, Exercise
  Manage/Create, Workout Recorder, Post Workout and Historical Analytics. The sidebar and
  the routine editor are done.
- [ ] Test on a real phone: drag-to-reorder, the sidebar drawer, number inputs, and the
  dialog on the recorder. Only desktop browsers and an emulated narrow viewport were used.
- [ ] Check browser back/forward after HTMX navigation (partial vs full page), and that
  the tab title is right for pages without an `<h1>`.
- [ ] A way to discard a stuck local session from the UI (today Cancel handles most cases).
- [ ] Optional: show a "last synced" indicator on the recorder while a workout is running.
- [ ] Credit the vendored libraries (Chart.js, SortableJS, HTMX; all MIT) in a notices file.

## Code quality

- [ ] Add tests for the `internal/web` handlers and the `internal/api` client (only routing
  and the live session are covered).
- [ ] Normalise line endings: files are written with CRLF, so `gofmt -l` lists every file.
  Add a `.gitattributes` and run `gofmt -w`.
- [ ] CI: run `go vet ./...` and `go test ./...` on push.
- [ ] The exercise list and exercise data are cached for 5 minutes; routines and the user
  are not. Revisit once the data source is direct.
