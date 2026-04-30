# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go port of the Ciholas Data Protocol (CDP) parser plus a UDP-multicast → NATS bridge binary, with optional 2D polygon geofencing on `PositionV3` updates. CDP is a little-endian UDP wire format from Ciholas RTLS hardware: a 20-byte header followed by zero or more `(type, size, payload)` data items. Reference Python implementations live at `../cdp-py` (parser) and `../cdp-geofencing` (geofencing).

Module path: `github.com/velociti/cdp-go` (Go 1.24).

## Commands

```bash
go build ./cmd/cdp-nats-bridge        # build the bridge binary
go test ./...                         # run all tests
go test ./pkg/cdp -run TestDecode     # run a single test by name
go vet ./...                          # static checks
```

Run the bridge:
```bash
./cdp-nats-bridge --config configs/example.yaml
./cdp-nats-bridge --group 239.255.76.67 --port 7667 --nats-url nats://localhost:4222
```

Watch the live feed (requires the `nats` CLI):
```bash
nats sub "cdp.>"
```

## Architecture

Three layers, deliberately kept separate:

**`pkg/cdp/`** — the public, reusable parser. Pure decoding only; no I/O, no transport. `Decode([]byte) (*Packet, error)` returns a `Packet` whose `Items` slice contains typed payloads (`*PositionV3`, `*AccelerometerV2`, etc.) or `*Unknown` for unregistered type IDs. Unknown types are not errors — they pass through with raw bytes preserved.

**`internal/geofence/`** — pure 2D-polygon engine: zone geometry, per-tag hysteresis state machine, event types. No NATS coupling — defines an `EventSink` interface that the bridge implements. The engine consumes `*cdp.PositionV3` directly and emits `geofence.Event` values.

**`internal/cdpbridge/`** — binary-specific glue: config loading, multicast listener, JSON publish, geofence wiring. Importable only by `cmd/cdp-nats-bridge`. The orchestrator is `Run(ctx, cfg)` in `run.go`: it connects to NATS, optionally builds a geofence engine when `cfg.Geofence.Enabled()`, spawns `listen()` on a goroutine, and pumps datagrams through `publish()` which calls `cdp.Decode`, emits one NATS message per data item, and feeds each `*PositionV3` to the engine.

**Shared infrastructure** under `internal/`:
- `config/` — `Broker` and `Logger` structs with yaml tags, plus `LoadYAMLInto` (uses `KnownFields(true)` so unknown YAML keys are errors)
- `broker/` — `Connect(cfg config.Broker) (*nats.Conn, error)`; owns the full nats.Option list
- `logger/` — slog setup from a `config.Logger`

## Adding a new CDP data item type

Two steps in `pkg/cdp/items.go`:

1. Define the struct with `json:` tags and a `decode<Name>(b []byte) (any, error)` function. Use the `u8`/`u16`/`u32`/`i32`/`u64`/`f32` helpers from `cdp.go` for little-endian reads. Return `errShort` if `len(b)` is below the fixed-size portion.
2. Register it in `init()`: `registry[0xNNNN] = registryEntry{"GoName", "subject_token", decodeGoName}`. The subject token is the lowercase, version-suffix-stripped name (e.g. `PositionV3` → `position`).

Then add a row to the table in `README.md`. The dispatch in `cdp.go:Decode` is registry-driven — no other code changes needed.

## Geofence package

`internal/geofence/`:
- `geometry.go` — `Point` (int32 mm, matches `cdp.PositionV3.X/Y`), `Zone`, `RGB`, `bbox` prefilter, and `PointInPolygon`. Uses W. R. Franklin's strict-inequality ray-cast in integer arithmetic (int64 intermediates), with no division and no epsilon. Replaces the float-equality + magic-epsilon hand-rolled math in `cdp-geofencing/zone.py`.
- `engine.go` — `Engine` holds zones + per-tag hysteresis state + an `EventSink`. **Not goroutine-safe** by design (the bridge serializes through one channel reader). `OnPosition(serial, *PositionV3)` is the entry point. Hysteresis semantics: a *pending* proposed state must hold for N consecutive packets before commit; oscillation never commits (corrected vs. the Python).
- `events.go` — `Event`, `EventType`, `EventSink` interface. The engine doesn't import NATS; the sink at `internal/cdpbridge/geofence_sink.go` does the publishing.
- `config.go` — `Config` with `yaml:"geofence"` tag, embedded in `cdpbridge.Config`. `Enabled() == false` (empty `Zones`) keeps the engine unconstructed and the per-packet path identical to a build without geofencing.

Boundary behavior: points exactly on a polygon edge or vertex are reported as inside/outside deterministically for a given input, but the choice is implementation-defined (not specified to match any particular convention). At UWB millimeter resolution this isn't physically meaningful; tests in `geometry_test.go` pin the chosen behavior so it can't drift silently.

Coordinate range: int32 millimeters throughout. Realistic UWB scenarios (≤ ±10⁸ mm = 100 km) are well within the int64-product safety bound; vertices at int32 extremes could overflow the cross-multiply but no real config gets close.

Subject construction: `<prefix>.<event_type>.<tag_serial_hex8>.<zone_slug>`. `zone_slug` comes from `slugifyZoneName()` (lowercase, NATS-illegal characters → `_`). Constructed in `internal/cdpbridge/geofence_sink.go`, not in the geofence package itself.

## Config resolution

Priority (lowest → highest): defaults → YAML (`--config` / `CONFIG_FILE`) → env vars → flags. Implemented in `internal/cdpbridge/config.go`: the YAML and env passes happen first, then flags are declared with the post-env values as their defaults so `flag.Parse` only overrides operator-provided flags. Env names are documented in the README's flag tables (e.g. `CDP_GROUP`, `NATS_URL`, `LOG_LEVEL`).

## NATS output format

**Per-item bridge output** (always on): `<prefix>.<subject_token>.<sender_serial_hex8>` where `sender_serial_hex8` is 8 lowercase hex chars with no colons (NATS subjects forbid them — `Serial.Hex()` produces this; `Serial.String()` produces the colon-separated display form `XX:XX:XXXX`). Body: `envelope` struct in `internal/cdpbridge/publish.go`: `{type, type_name, packet:{sequence, sender_serial}, data:<typed payload>}`. `type` carries the original CDP type ID with version (e.g. `0x0135`) so consumers can disambiguate items whose subject token had the version suffix stripped.

**Geofence events** (only when `cfg.Geofence.Enabled()`): `<geofence_prefix>.<enter|exit>.<tag_serial_hex8>.<zone_slug>`. Body: `geofence.Event` JSON. The geofence stream is independent from and parallel to the per-item bridge stream — both run from the same `publish()` call site.

Per-datagram decode errors abort that datagram; per-item publish errors are logged and the loop continues with remaining items.
