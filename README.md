# MyTurn

A tiny local-network web app that shows a household/group whose turn it is
for a daily rotating responsibility. Works with any number of members. No
cloud services, no accounts, no database — just Go's standard library, a
JSON file, locally hosted Bootstrap CSS, and vanilla JavaScript.

## Quick start

```
go run .
```

or build a binary:

```
go build -o myturn .
./myturn
```

By default the server listens on `0.0.0.0:8031`. On startup it prints the
URLs you can use from any device on the same Wi-Fi network, e.g.:

```
MyTurn running on:
  http://localhost:8031
  http://192.168.1.42:8031
```

Open that LAN URL from an iPhone (or any) browser. Everyone on the same
Wi-Fi network sees the same rotation.

### Changing the port

```
PORT=9090 go run .
```

### Config file location

The app reads/writes `data/config.json` relative to the working directory by
default. Override with `MYTURN_CONFIG=/path/to/config.json`. If the file
does not exist, a default config with a few placeholder members is created
automatically.

`data/config.json` is gitignored since it holds your real group's names —
copy [data/config.example.json](data/config.example.json) to
`data/config.json` and edit it (or just let the app create a default on
first run) rather than committing your real data.

### Admin secret

Settings access is gated by a shared secret, set via:

```
MYTURN_ADMIN_SECRET=your-secret-here go run .
```

If unset, it falls back to a default (`changeme`) and logs a warning on
startup — fine for quick local testing, but set a real value before leaving
the server running.

## Running with Docker

Build the image, tagged with the version from the [VERSION](VERSION) file:

```
docker build --build-arg VERSION=$(cat VERSION) -t myturn:$(cat VERSION) .
```

The version is also embedded in the binary (printed on startup, e.g.
`MyTurn 0.1.0 running on:`) and set as the `org.opencontainers.image.version`
image label. Bump [VERSION](VERSION) before building a new release tag.

Run it, keeping the config persisted in a named volume so it survives
container restarts/upgrades. The image defaults to port 8031 (handy if you
already have other containers on 8080, 8000, 9090, etc.):

```
docker run -d --name myturn -p 8031:8031 \
  -e MYTURN_ADMIN_SECRET=your-secret-here \
  -v myturn-data:/data myturn:$(cat VERSION)
```

The image stores its config at `/data/config.json` (set via `MYTURN_CONFIG`)
and creates a default one on first run if the volume is empty. To use a
different port, set `PORT` and republish accordingly, e.g.:

```
docker run -d --name myturn -p 9090:9090 -e PORT=9090 -v myturn-data:/data myturn:$(cat VERSION)
```

To use a host directory instead of a named volume, make sure it's writable
by the container's non-root user (uid `10001`):

```
mkdir -p ./data && chown 10001:10001 ./data
docker run -d --name myturn -p 8031:8031 -v "$PWD/data:/data" myturn:$(cat VERSION)
```

## Using a friendly hostname (optional)

Instead of typing the LAN IP every time, you can set up a local hostname
such as `myturn.local`:

- **macOS/iOS (Bonjour)**: if the server machine is a Mac, its Bonjour name
  (`<hostname>.local`) is usually already reachable from iPhones on the same
  network — no extra setup needed if the hostname is memorable.
- **Linux**: install and enable `avahi-daemon` to broadcast `<hostname>.local`
  via mDNS.
- **Router/local DNS**: some routers let you assign a custom LAN hostname to
  a device's static IP via DHCP reservation settings.

This project intentionally does **not** bundle an mDNS library — it's not
required, and keeping dependencies at zero keeps the project simple to
read and build. Use whichever OS/router mechanism is convenient, or just use
the LAN IP printed at startup.

## How the rotation works

The config file (`data/config.json`) looks like:

```json
{
  "group_name": "Home",
  "timezone": "Europe/Madrid",
  "start_date": "2026-10-01",
  "members": ["Ana", "Bruno", "Carla", "Diego", "Elena"],
  "skip_dates": []
}
```

For any date, the default (calculated) person is:

```
daysSinceStart = date - start_date   (in whole calendar days)
index          = ((daysSinceStart % N) + N) % N
currentPerson  = members[index]
```

This is computed on the fly from `start_date` — the JSON file is **not**
rewritten every day just to advance the rotation. The current date is
always derived using the configured `timezone`, not the server's local time
or UTC.

Use **Saltar el turno de hoy** in Settings to mark today as a no-turn day.
The person scheduled for today moves to tomorrow, and all subsequent turns
shift forward by one day. Skip dates are persisted in `skip_dates`; repeating
the action on the same day does not shift the schedule again.

## Administration

Open the ⚙️ **Settings** link from the main screen. Settings requires the
shared `MYTURN_ADMIN_SECRET`. From there you can:

- change the group name
- add/remove/rename/reorder members
- change the start date and timezone
- skip today's turn, shifting the schedule forward by one day

All changes are validated server-side and written atomically (temp file +
rename) to avoid corrupting the config file, even under concurrent updates
from multiple phones.

## HTTP API

| Method & path      | Description                              |
|---------------------|-------------------------------------------|
| `GET /api/state`    | Today/tomorrow/upcoming rotation info      |
| `GET /api/config`    | Current raw configuration                  |
| `PUT /api/config`    | Replace the full configuration             |
| `POST /api/skip/today` | Skip today's turn (requires admin secret) |

All responses are JSON with `Content-Type: application/json`.

## PWA / Home Screen install

The app ships a `manifest.json` and a service worker that caches the static
HTML/CSS/JS shell, so the page can open if the server is temporarily
unreachable. API responses are not cached; without the server, the home page
shows a server-unavailable message instead of stale turn information. The
Bootstrap theme follows the device's light/dark preference.

On iPhone: open the LAN URL in Safari → Share → **Add to Home Screen**.

## Project layout

```
myturn/
  go.mod
  main.go                     entrypoint, embeds web/, LAN IP discovery, graceful shutdown
  internal/
    config/config.go          load/save/validate JSON config, atomic writes
    rotation/rotation.go      rotation calculation
    rotation/rotation_test.go unit tests
    httpserver/handlers.go    JSON API + static file serving
  web/                        embedded Bootstrap frontend and PWA assets
  data/config.json            external, editable configuration (not embedded)
  Dockerfile                  multi-stage build (golang -> alpine) for containerized runs
  .dockerignore
```

## Tests

```
go test ./...
```

Covers: first day picks the first member, subsequent days advance, wrapping
after the last member, dates before `start_date`, leap-day arithmetic, and
timezone handling.

## Concurrency & safety

In-memory config is protected by a `sync.RWMutex` in the HTTP server. Writes
go through `config.Save`, which validates input, then writes a temp file and
renames it over `data/config.json`, so concurrent requests can't corrupt the
file or leave it half-written.
