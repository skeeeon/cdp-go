# cdp-go

Go port of the Ciholas Data Protocol (CDP) parser, plus a small bridge binary
that listens for CDP UDP multicast traffic, decodes each packet, and publishes
the decoded data items as JSON onto NATS.

The reference Python implementation is [`cdp-py`](../cdp-py).

## Layout

```
pkg/cdp/                       # public reusable CDP parser
internal/
  config/                      # shared config types (Broker, Logger) + YAML loader
  broker/                      # NATS connect helper (consumes config.Broker)
  logger/                      # slog setup (consumes config.Logger)
  cdpbridge/                   # binary-specific: config+listener+publish+run
cmd/
  cdp-nats-bridge/             # tiny entry point
configs/                       # example.yaml (config file)
```

The `internal/config`, `internal/broker`, and `internal/logger` packages are
shared infrastructure — when a second NATS-publishing binary is added (e.g. a
geofence source), it lives in its own `internal/<name>/` and reuses these.

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
| `--log-level` | `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

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
| `--nats-drain-timeout` | `NATS_DRAIN_TIMEOUT` | `30s` | |
| `--nats-flush-timeout` | `NATS_FLUSH_TIMEOUT` | `5s` | |
| `--nats-no-echo` | `NATS_NO_ECHO` | `false` | Suppress own messages |

## Testing

```bash
go test ./...
```

Watch the live feed:
```bash
nats sub "cdp.>"
```

## License

Creative Commons Attribution 4.0 International, matching the Python parser.
