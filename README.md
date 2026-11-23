# dfmpd-playback-mpv

Independent experimental Playback plugin for DFMPD. Plugin ID (the protocol's
capability `name`) and executable name are both `dfmpd-playback-mpv`.
It controls a child mpv process through mpv JSON IPC; its host connection is the
separate DFMPD SDK stdio protocol. It owns no queue, Source, or client IPC.

## Requirements and build

- Go 1.27.2 or later, as specified by go.mod.
- Linux for the currently tested DFMPD process-group supervision, and Unix sockets.
- mpv 0.41.0 is the verified version. Older versions have not been qualified;
  there is no claim of a lower supported minimum or runtime version gate.
  mpv must support `--input-ipc-server`, `loadfile`, `seek absolute+exact`,
  idle/pause/time-pos properties, and file-loaded/end-file events.
- mpv must include the codecs/output backend required by your media and system.
  FLAC and MP3 decoding belong to mpv, not this plugin.
- The sole Go module dependency is the published `github.com/dfmpd/dfmpd-sdk`
  pseudo-version pinned in go.mod. No daemon source, go.work, or local replace is
  required. FFmpeg is only needed by the host's generated-FLAC integration tests.

```sh
GOWORK=off go mod tidy
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
GOWORK=off go build ./...
GOWORK=off go build -o ./dfmpd-playback-mpv ./cmd/dfmpd-playback-mpv
```

Tests use fake IPC peers plus real mpv with generated WAV/null output. Real mpv
cases skip when mpv is absent. The internal adapter and tests were copied from
DFMPD for C2; its local types are converted to SDK types only at the plugin
boundary in `internal/playback/plugin`.

## Run with DFMPD

Build/install the daemon separately, then provide the explicit plugin path:

```sh
dfmpd -playback-plugin /absolute/path/dfmpd-playback-mpv
# Optional mpv executable override (forwarded to the plugin in C2):
dfmpd -playback-plugin /absolute/path/dfmpd-playback-mpv -mpv /usr/bin/mpv
```

The daemon requires a private `XDG_RUNTIME_DIR`. Use `dfmpc` separately for
queue add/play, next/previous, pause/resume, seek, stop, and status.
The plugin also accepts `-mpv executable` directly, but it is not an interactive
player: stdin/stdout are reserved for newline-delimited JSON-RPC, stderr for
logs. The host must initialize the session and supervise the process.

Protocol version 1 declares kind `playback`, input `local-file`, and operations
play/pause/resume/stop/seek/status. The SDK handles `plugin.initialize` and
`plugin.shutdown`, framing, required-field validation and deadlines. Unknown
JSON fields are allowed within the supported version. Public DFMPD client IPC
and raw mpv replies are not exposed through this boundary.

## Limits and cleanup

Only accessible local regular files are accepted. There is no URL/authentication
support, automatic discovery, installation, restart, or hot reload. A successful
play acknowledges acceptance, not audible playback. Timeouts/lost replies can
leave an operation outcome unknown; mutations must not be automatically retried.
The SDK protocol remains experimental, not a stable general plugin API.

The plugin normally closes/reaps mpv and removes its private socket directory on
shutdown, EOF, or SIGINT/SIGTERM. SIGKILL cannot run this cleanup. DFMPD's host-side
supervisor owns process-group TERM/KILL cleanup after plugin death/stalls; the
plugin does not create a new group or escape the host's group. The host reaps its
direct child, while init/systemd reaps orphaned mpv descendants after a plugin
crash. Daemon SIGKILL, escaped groups, unavailable signal permissions, and
uninterruptible kernel sleep remain outside bounded cleanup guarantees. A killed
plugin can leave its private socket directory behind. This is not a sandbox.

C3 removes the daemon's duplicate implementation and moves `cmd/mpv-probe` here.
Build/run the diagnostic with `go run ./cmd/mpv-probe AUDIO_FILE` (optional
`-mpv executable` before the file). Its command/output behavior is preserved.
For C3 hosts the backend override is explicit generic argument forwarding:

```sh
dfmpd -playback-plugin /absolute/path/dfmpd-playback-mpv \
 -playback-plugin-arg=-mpv -playback-plugin-arg=/usr/bin/mpv
```

The earlier daemon `-mpv` example applies only to hosts through C2.

## Host compatibility tests

From a separate DFMPD checkout, without requiring this repository's placement:

```sh
GOWORK=off DFMPD_TEST_PLAYBACK_PLUGIN=/absolute/path/dfmpd-playback-mpv \
  go test ./cmd/dfmpd ./internal/playback/plugin \
  -run 'TestRealFLACQueue|TestRealMPVFailureRecovery' -count=1 -v
```

These verify two FLAC queue entries, next/previous, pause/resume, seek/stop,
normal process/socket cleanup, SIGKILL/SIGSTOP recovery, and continued Core/Queue
availability. They use null audio, not audible output. On C3 hosts these media cases skip when the variable is absent; normal host
tests use fake SDK processes. C2 hosts retain their original in-tree default.

Repository: https://github.com/dfmpd/dfmpd-playback-mpv.
This repository is managed independently of the DFMPD daemon.
Code copied from DFMPD retains AGPL-3.0 licensing; see LICENSE.
