# TODO

What is left after merging the backend into this workspace. Grouped by area; the first
group is what stands between this repo and retiring the old apps.

## Before the old apps can be retired

- [ ] **Automated deploy (no Docker).** The single binary runs under the existing systemd
  service on the VM; `SESSION_FILE` and other temporary data are plain local files.
  - [ ] Adopt a `master` + `dev` branch flow: work lands on `dev` and is tested there, then
    merged into `master`.
  - [ ] GitHub Actions workflow on push to `master`: build with `CGO_ENABLED=0`
    (`GOOS=linux`, `GOARCH=arm64` for the Orange Pi), `scp` the binary to the VM, then
    restart the systemd service over SSH. Store the SSH key, host and user as repository secrets.
  - [ ] Also a local script (same build + `scp` + restart) for manual pushes.
  - [ ] Make the upload safe: copy to a temp name and `mv` into place, so a running binary
    is never overwritten mid-copy.
  - [ ] Decide `BIND_ADDR` for the new deployment (public HTTPS now, so likely `:443`/`:80`
    rather than a Tailscale IP), and where the env/`.env`
    and `SESSION_FILE` live on the VM (the service's `EnvironmentFile` / `WorkingDirectory`).
- [ ] **HTTPS certificate support with `golang.org/x/crypto/acme/autocert`.** Serve the UI
  over TLS at `https://fitness-tracker.the-tunnster.com` (no port in the URL; standard 443),
  with certificates issued and renewed automatically (Let's Encrypt). ddclient on the VM
  already keeps the domain pointing at it.
  - [ ] Add config (e.g. `TLS_DOMAIN=fitness-tracker.the-tunnster.com`, `CERT_CACHE_DIR`,
    optional `ACME_EMAIL`); keep plain HTTP when unset so local development still works.
  - [ ] Use `autocert.Manager` with `HostPolicy` limited to the configured domain and a
    `DirCache` in a local directory that deploys never overwrite.
  - [ ] Serve on 443 with `manager.TLSConfig()`, and run a port 80 listener
    (`manager.HTTPHandler`) for the HTTP-01 challenge and redirect to HTTPS.
  - [ ] Let the service bind ports 80/443 without root (systemd `AmbientCapabilities=CAP_NET_BIND_SERVICE`).
  - [ ] Open/forward ports 80 and 443 to the VM (router/firewall) and check they are
    reachable from the internet, which Let's Encrypt needs for HTTP-01.
  - [ ] Basic Auth is already in place (`AUTH_USER`/`AUTH_PASSWORD`), but it must not be
    reachable over plain HTTP: do not open ports 80/443 until HTTPS works, and make the
    port 80 listener only redirect.
- [ ] **Confirm feature parity with Streamlit**, page by page, using real data.
- [ ] **Remove the Streamlit app** (`fitness-frontend`) once parity is confirmed. Until
  then it keeps working if started with `BACKEND_URL=http://opi5:8090/api` (the API moved
  under `/api`). Do not delete it without an explicit go-ahead.
- [ ] **Stop the old services on the Orange Pi** (Streamlit on 8501, the old backend on 8090)
  when the new binary takes over (it moves to 80/443, so 8090 is freed).
- [ ] **Rotate the Atlas database password.** It was pasted into a chat while testing.
- [ ] **Try a real workout end to end** with the live-session file on real data, including
  killing the server mid-workout and restarting it.

## Architecture

- [ ] **Replace the loopback HTTP hop with direct calls** to the `database`/`service`
  packages (put an interface between `internal/web` and the data source). Then drop
  `internal/api`'s HTTP client, and have the live session push straight to the database.
  - [ ] Decide what happens to `MOCK_ANALYTICS` (keep as a second implementation of the
    same interface, or drop it once real data exists).
- [ ] **Migrate to `go.mongodb.org/mongo-driver/v2`.** `go get` warns that v1 is deprecated
  (still works). Do it after the first database test pass, as a separate change: `primitive.ObjectID`
  becomes `bson.ObjectID`, `mongo.Connect` drops the context, and some `options` change (~22 files).
- [ ] Decide the final module path (currently `fitness-tracker`, the backend's) and the
  repo name / remote. There is no git remote yet.
- [ ] Basic Auth now covers the UI and `/api` on the one listener. Later: login rate
  limiting / lockout, a proper login page with sessions (or OIDC), and whether `/api` still
  needs to be reachable from outside the process at all (see the direct-calls item above).

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
