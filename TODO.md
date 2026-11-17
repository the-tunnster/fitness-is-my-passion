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
  - [ ] Reuse `scripts/deploy.sh` (local test + build + upload + restart, already safe and
    with `--rollback`) from the workflow instead of duplicating its steps.
  - [ ] Decide where the env/`.env`
    and `SESSION_FILE` live on the VM (the service's `EnvironmentFile` / `WorkingDirectory`).
- [ ] **Stop the old services on the Orange Pi** (Streamlit on 8501, the old backend on 8090)
  when the new binary takes over (it keeps using 8090). The JSON API (`/api`) no longer
  exists, so the Streamlit app stops working; delete `fitness-frontend` when you're ready.
- [ ] **Rotate the Atlas database password.** It was pasted into a chat while testing.

## Architecture

- [ ] Decide the final module path (currently `fitness-tracker`, the backend's) and the
  repo name / remote. There is no git remote yet.
- [ ] Rename `internal/api`: it is now the UI's data source, not an API client.

## Backend review items not fixed yet

- [ ] No tests cover the `database` package or `api.DB`, which need MongoDB. Needs a test
  database or a fake behind an interface (see Architecture).

## UI

- [ ] **Visual and spacing pass** on the pages not reworked yet: Home, Profile, Exercise
  Manage/Create, Workout Recorder, Post Workout and Historical Analytics. The sidebar and
  the routine editor are done.
- [ ] Test on a real phone: drag-to-reorder, the sidebar drawer, number inputs, and the
  dialog on the recorder. Only desktop browsers and an emulated narrow viewport were used.
- [ ] Check browser back/forward after HTMX navigation (partial vs full page), and that
  the tab title is right for pages without an `<h1>`.
- [ ] A way to discard a stuck local session from the UI (today Cancel handles most cases).
- [ ] Optional: show a "last synced" indicator on the recorder while a workout is running.
- [ ] Credit the vendored libraries (Chart.js, SortableJS, HTMX; all MIT) and the Material
  Symbols font (Apache 2.0) in a notices file.

## Code quality

- [ ] Add tests for the `internal/web` handlers and `api.DB` (only routing
  and the live session are covered).
- [ ] Normalise line endings: files are written with CRLF, so `gofmt -l` lists every file.
  Add a `.gitattributes` and run `gofmt -w`.
- [ ] CI: run `go vet ./...` and `go test ./...` on push.
- [ ] The exercise list and exercise data are cached for 5 minutes; routines and the user
  are not. Revisit once the data source is direct.

## Later

- [ ] **Migrate to `go.mongodb.org/mongo-driver/v2`.** `go get` warns that v1 is deprecated
  (still works). Not urgent; do it later as a separate change, once the above is settled: `primitive.ObjectID`
  becomes `bson.ObjectID`, `mongo.Connect` drops the context, and some `options` change (~22 files).
