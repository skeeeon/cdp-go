# cdp-go

Go port of the Ciholas Data Protocol (CDP) parser, plus a small bridge binary
that listens for CDP UDP multicast traffic, decodes each packet, and publishes
the decoded data items as JSON onto NATS. An optional 2D polygon geofencing
feature emits zone enter/exit events for each tag's `PositionV3` updates.

The reference Python implementations are [`cdp-py`](../cdp-py) (parser) and
[`cdp-geofencing`](../cdp-geofencing) (geofencing).

## Layout

```
pkg/cdp/                       # public reusable CDP parser
internal/
  config/                      # shared config types (Broker, Logger) + YAML loader
  broker/                      # NATS connect helper (consumes config.Broker)
  logger/                      # slog setup (consumes config.Logger)
  geofence/                    # 2D polygon engine: zones, hysteresis, events
  cdpbridge/                   # binary-specific: config+listener+publish+run
cmd/
  cdp-nats-bridge/             # tiny entry point
configs/                       # example.yaml (config file)
```

The `internal/config`, `internal/broker`, and `internal/logger` packages are
shared infrastructure. `internal/geofence` is pure logic (no NATS coupling) —
the bridge wires it to NATS via a small sink in `internal/cdpbridge/`.

## Supported data items

A common subset is decoded into typed Go structs. Anything else passes
through as a fallback `Unknown` containing the raw payload bytes.

| Type ID | Struct | Subject token |
|---|---|---|
| 0x0135 | PositionV3 | `position` |
| 0x0136 | AnchorPositionStatusV3 | `anchor_position_status` |
| 0x0137 | DeviceActivityState | `device_activity_state` |
| 0x0138 | DeviceHardwareStatusV2 | `device_hardware_status` |
| 0x0139 | AccelerometerV2 | `accelerometer` |
| 0x013A | GyroscopeV2 | `gyroscope` |
| 0x013B | MagnetometerV2 | `magnetometer` |
| 0x013C | PressureV2 | `pressure` |
| 0x013D | QuaternionV2 | `quaternion` |
| 0x013E | TemperatureV2 | `temperature` |
| 0x013F | DeviceNames | `device_names` |
| 0x0140 | Synchronization | `synchronization` |
| 0x0141 | RoleReport | `role_report` |
| 0x0148 | UserDefinedV2 | `user_defined` |
| 0x0149 | NetworkTime | `network_time` |
| 0x014A | AnchorHealthV5 | `anchor_health` |
| 0x015A | NtRealTimeMappingV1 | `nt_realtime_mapping` |
| 0x0160 | BootloadProgress | `bootload_progress` |
| 0x0164 | PolarCoordinatesV1 | `polar_coordinates` |
| 0x0171 | ImageDiscoveryV2 | `image_discovery` |
| (other) | Unknown | `unknown_<typeid_hex4>` |

Adding a new type: append a struct + decoder in `pkg/cdp/items.go` and register
it in the `init()` function at the top of that file.

## Build & run

```bash
go build ./cmd/cdp-nats-bridge

# With a config file:
./cdp-nats-bridge --config configs/example.yaml

# Or pure flags:
./cdp-nats-bridge \
    --group 239.255.76.67 --port 7667 \
    --nats-url nats://localhost:4222
```

Use `--help` to see every flag.

## NATS subjects

```
<prefix>.<subject_token>.<sender_serial_hex8>
```

- `prefix` defaults to `cdp` (override via `--prefix` / `CDP_NATS_PREFIX`).
- `subject_token` is the lowercase, version-stripped name from the table above.
- `sender_serial_hex8` is the 8-character lowercase hex of the packet header
  serial — no colons, since NATS subjects forbid them.

Examples:
```
cdp.position.01234567
cdp.anchor_health.deadbeef
cdp.unknown_abcd.01020304
```

## Published JSON envelope

Each NATS message body is a JSON object:

```json
{
  "type": "0x0135",
  "type_name": "PositionV3",
  "packet": {
    "sequence": 12345,
    "sender_serial": "01:23:4567"
  },
  "data": {
    "serial_number": "01:23:4567",
    "network_time": 100,
    "x": 10,
    "y": 20,
    "z": 30,
    "quality": 9000,
    "anchor_count": 4,
    "flags": 0,
    "smoothing": 5
  }
}
```

`type` and `type_name` carry the original CDP type ID (with version) so a
consumer can disambiguate items whose subject token has the version
suffix stripped.

## Geofencing (optional)

When a `geofence:` block is present in the YAML config, every `PositionV3`
the bridge sees is also tested against a list of 2D polygon zones. Committed
zone-membership transitions are published as JSON events on a parallel NATS
subject tree. With no `geofence:` block (or an empty `zones:` list), the
feature is disabled and the bridge runs identically to a build without it.

Coordinates are int32 millimeters — the same units `PositionV3` uses on the
wire. (Deliberate divergence from the Python implementation, which used
floats. Integer mm eliminates float-comparison foot-guns.)

Zones are 2D polygons defined by ≥3 vertices. The polygon is implicitly
closed (last vertex connects to first). Vertex order is irrelevant
(orientation-agnostic).

```yaml
geofence:
  hysteresis: 5    # consecutive packets a new zone state must hold before commit; 0|1 = immediate
  prefix: geofence
  zones:
    - name: Paper
      vertices: [[0, 0], [10000, 0], [10000, 10000], [0, 10000]]
      rgb: [255, 255, 0]
    - name: Loading Dock
      vertices: [[20000, 0], [30000, 0], [25000, 10000]]
      rgb: [0, 0, 255]
```

### Hysteresis

`hysteresis: N` means "a proposed new zone-membership state must hold for N
consecutive `PositionV3` packets from the same tag before it is committed."
A tag oscillating between two zones every packet never commits — the
proposed state has to be *stable*, not just different. `hysteresis: 0` and
`hysteresis: 1` both commit immediately on any observed change.

### NATS subjects

```
<prefix>.<event_type>.<tag_serial_hex8>.<zone_slug>
```

- `prefix` defaults to `geofence` (override via `--geofence-prefix` / `GEOFENCE_PREFIX`)
- `event_type` is `enter` or `exit`
- `tag_serial_hex8` matches the bridge's tag rendering (8 lowercase hex, no colons)
- `zone_slug` is the lowercased zone name with NATS-illegal characters
  (`.`, `>`, `*`, whitespace) replaced by underscores

Useful wildcards:

```
geofence.>                       # all geofence events
geofence.enter.>                 # all enters
geofence.*.*.paper               # all transitions touching the "paper" zone
geofence.*.01020304.>            # all transitions for one tag
```

### Geofence event JSON

```json
{
  "type": "enter",
  "tag": "01:02:0304",
  "zone": "Paper",
  "in_zones": ["Paper"],
  "network_time": 12345678901234,
  "position": {"x": 500, "y": 700},
  "color": {"r": 255, "g": 255, "b": 0}
}
```

`in_zones` is the full committed zone set *after* the transition (sorted
alphabetically), so a downstream consumer never has to maintain its own
state to answer "where is this tag now". When a tag enters or exits multiple
zones simultaneously (overlapping zones, or jumping across non-adjacent
ones), one event is emitted per zone touched, with exits ordered before
enters.

### Geofence flags

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--geofence-prefix` | `GEOFENCE_PREFIX` | `geofence` | NATS subject prefix |
| `--geofence-hysteresis` | `GEOFENCE_HYSTERESIS` | `5` | Consecutive-packet count |
| `--geofence-tag-ttl` | `GEOFENCE_TAG_TTL` | `1h` | Drop per-tag state if no position update for this long; `0` disables |

Zones are YAML-only (lists don't have a sensible flag form).

`tag_ttl` is wall-clock time, **not** CDP NetworkTime. NetworkTime is a UWB
sync clock that resets on tag reboot, so it's the wrong source for "I
haven't seen this tag in a while" decisions.

## Configuration

Resolved in priority order (lowest to highest):

1. Built-in defaults
2. YAML file at `--config <path>` (or `CONFIG_FILE` env)
3. Environment variables
4. Command-line flags

See [configs/example.yaml](configs/example.yaml) for the full file shape.
Unknown YAML keys are an error (catches typos early).

### CDP listener

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--group` | `CDP_GROUP` | `239.255.76.67` | Multicast group |
| `--port` | `CDP_PORT` | `7667` | UDP port |
| `--iface` | `CDP_INTERFACE` | (auto) | Network interface name |
| `--prefix` | `CDP_NATS_PREFIX` | `cdp` | NATS subject prefix |
| `--udp-read-buffer` | `CDP_UDP_READ_BUFFER` | `1048576` (1 MiB) | UDP socket SO_RCVBUF size in bytes |
| `--log-level` | `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

**UDP buffer tuning:** if `netstat -su` shows `RcvbufErrors` climbing, raise
`--udp-read-buffer`. Values above ~1 MiB usually require a sysctl bump
(`sudo sysctl -w net.core.rmem_max=8388608`) — the kernel silently caps
`SetReadBuffer` at `rmem_max`.

### NATS

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--nats-url` | `NATS_URL` | `nats://localhost:4222` | Server URL(s) — comma-separated allowed |
| `--nats-name` | `NATS_NAME` | `cdp-nats-bridge` | Connection name |
| `--nats-user` | `NATS_USER` | | Username (paired with `--nats-password`) |
| `--nats-password` | `NATS_PASSWORD` | | Password |
| `--nats-token` | `NATS_TOKEN` | | Auth token |
| `--nats-creds` | `NATS_CREDS_FILE` | | JWT/NKey credentials file |
| `--nats-nkey` | `NATS_NKEY_SEED_FILE` | | NKey seed file |
| `--nats-tls-ca` | `NATS_TLS_CA` | | TLS CA bundle |
| `--nats-tls-cert` | `NATS_TLS_CERT` | | TLS client cert |
| `--nats-tls-key` | `NATS_TLS_KEY` | | TLS client key |
| `--nats-tls-insecure` | `NATS_TLS_INSECURE` | `false` | Skip TLS verification (dev only) |
| `--nats-max-reconnects` | `NATS_MAX_RECONNECTS` | `-1` | `-1` = forever |
| `--nats-reconnect-wait` | `NATS_RECONNECT_WAIT` | `2s` | |
| `--nats-reconnect-jitter` | `NATS_RECONNECT_JITTER` | `100ms` | |
| `--nats-ping-interval` | `NATS_PING_INTERVAL` | `2m` | |
| `--nats-max-pings-out` | `NATS_MAX_PINGS_OUT` | `2` | |
| `--nats-flush-timeout` | `NATS_FLUSH_TIMEOUT` | `5s` | Blocks shutdown until pending publishes flush, or this fires |
| `--nats-no-echo` | `NATS_NO_ECHO` | `false` | Suppress own messages |

## Testing

```bash
go test ./...
```

Watch the live feeds:
```bash
nats sub "cdp.>"           # decoded data items
nats sub "geofence.>"      # zone enter/exit events (when geofencing is enabled)
```

## License

Creative Commons Attribution 4.0 International, matching the Python parser.
