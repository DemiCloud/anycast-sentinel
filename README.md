# anycast-sentinel

A stateless, one-shot health-gated anycast announcer. On each invocation it
evaluates a list of health checks and either adds or removes an anycast IP
address on a network interface using netlink. Systemd timers control execution
frequency; the sentinel itself never loops.

All checks must pass (AND semantics). Any single failure immediately withdraws
the address.

---

## Table of Contents

- [Features](#features)
- [Usage](#usage)
  - [run](#run)
  - [install](#install)
  - [uninstall](#uninstall)
  - [version](#version)
- [Installation](#installation)
- [Uninstallation](#uninstallation)
- [Configuration](#configuration)
  - [Full example](#full-example)
  - [[general] fields](#general-fields)
  - [[[checks]] fields](#checks-fields)
- [Systemd Units](#systemd-units)
  - [Service template](#service-template)
  - [Timer template](#timer-template)
  - [Drop-in overrides](#drop-in-overrides)
- [Output](#output)
- [Operational Notes](#operational-notes)
- [Building](#building)
- [Tests](#tests)
- [License](#license)

---

## Features

- Stateless one-shot execution — no counters, no state files
- Three health check types: systemd unit state, TCP connect, shell command
- AND semantics: all checks must pass or the address is withdrawn
- IPv4 `/32` and IPv6 `/128` support via netlink (dual-stack or single-stack)
- Dry-run mode: evaluates checks and reports decisions without touching routes
- Deterministic, grep-friendly output
- `install` subcommand installs hardened systemd template units and enables a timer instance
- `uninstall` subcommand disables a timer instance and cleans up unit files
- Operator overrides supported via systemd drop-ins

---

## Usage

```
anycast-sentinel <subcommand> [flags]

Subcommands:
  run        Evaluate health checks and manage the anycast address
  install    Install systemd templates and enable a timer instance
  uninstall  Disable a timer instance and remove the template unit files
  version    Show version information
  help       Show help for a subcommand
```

Run `anycast-sentinel help <subcommand>` or `anycast-sentinel <subcommand> --help`
for subcommand usage.

### run

```
anycast-sentinel run --config <path> [flags]

Flags:
  -c, --config    Path to configuration file (required)
      --dry-run   Evaluate checks but do not modify routes
  -h, --help      Show this help
```

Evaluates all configured health checks and adds or removes the anycast address
accordingly. Exits immediately after the routing decision.

### install

```
anycast-sentinel install <instance> [flags]

Arguments:
  <instance>     Instance name — enables anycast-sentinel@<instance>.timer

Flags:
      --interval    Timer firing interval (default: 5s)
      --boot-delay  Delay before first run after boot (default: 30s)
  -h, --help        Show this help
```

Installs systemd template units, reloads the daemon, enables and starts the
timer instance for `<instance>`. Creates `/etc/anycast/<instance>.toml` with
a sample configuration if it does not already exist.

### uninstall

```
anycast-sentinel uninstall <instance>

Arguments:
  <instance>     Instance name — disables anycast-sentinel@<instance>.timer
```

Disables and stops the named timer instance, removes the shared template unit
files (`anycast-sentinel@.service` and `anycast-sentinel@.timer`), and reloads
the systemd daemon. The instance config file (`/etc/anycast/<instance>.toml`)
is intentionally left in place.

### version

```
anycast-sentinel version
```

Prints version, commit, build date, builder, Go version, and OS/arch.

---

## Installation

Build and install the binary, then run `install` as root:

```sh
make build
sudo make install          # installs to /usr/local/sbin/anycast-sentinel
sudo anycast-sentinel install myservice
```

`PREFIX` and `DESTDIR` are supported for packaging:

```sh
make install PREFIX=/usr DESTDIR=/tmp/pkg
```

The `install` subcommand:

1. Creates `/etc/anycast/` if it does not exist
2. Renders and writes (only if changed):
   - `/etc/systemd/system/anycast-sentinel@.service`
   - `/etc/systemd/system/anycast-sentinel@.timer`
3. Writes `/etc/anycast/myservice.toml` with a sample config (skipped if already present)
4. Runs `systemctl daemon-reload`
5. Runs `systemctl enable --now anycast-sentinel@myservice.timer`

Edit the generated config file, then the timer will handle the rest.

## Uninstallation

```sh
sudo anycast-sentinel uninstall myservice
```

This will:

1. Disable and stop `anycast-sentinel@myservice.timer`
2. Remove `/etc/systemd/system/anycast-sentinel@.service`
3. Remove `/etc/systemd/system/anycast-sentinel@.timer`
4. Run `systemctl daemon-reload`

The config file `/etc/anycast/myservice.toml` is not removed.

---

## Configuration

Configuration files live in `/etc/anycast/<instance>.toml`. The file format is
[TOML](https://toml.io). Unknown fields are rejected.

### Full example

```toml
[general]
dev = "eth0"
ip4 = "203.0.113.10"   # IPv4 /32 to announce
ip6 = "2001:db8::1"    # IPv6 /128 to announce (optional)

# All checks must pass (AND semantics). At least one check is required.

# Check that a systemd unit is active
[[checks]]
name = "my-service"
type = "systemd"
unit = "my-service.service"

# Check that a TCP port is reachable
[[checks]]
name = "my-service-port"
type = "tcp"
host = "127.0.0.1"
port = 8080
timeout = "500ms"   # optional, default 500ms

# Check that a command exits 0
[[checks]]
name = "custom-check"
type = "command"
command = "/usr/local/bin/my-healthcheck.sh"
timeout = "5s"      # optional, kills the command if exceeded
```

### `[general]` fields

| Field | Required | Description |
|-------|----------|-------------|
| `dev` | yes | Network interface to bind the anycast address to |
| `ip4` | one of ip4/ip6 | IPv4 address to announce as a `/32` |
| `ip6` | one of ip4/ip6 | IPv6 address to announce as a `/128` |

### `[[checks]]` fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Human-readable label used in log output (required) |
| `type` | string | `systemd`, `tcp`, or `command` (required) |
| `unit` | string | Systemd unit name — `systemd` checks only |
| `host` | string | Hostname or IP — `tcp` checks only |
| `port` | int | TCP port number — `tcp` checks only |
| `command` | string | Shell command to run — `command` checks only |
| `timeout` | string | Duration string e.g. `500ms`, `5s` — `tcp` and `command` only |

`timeout` on a `systemd` check is a configuration error.

---

## Systemd Units

### Service template

```ini
[Unit]
Description=Anycast Sentinel (%i)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/sbin/anycast-sentinel run --config /etc/anycast/%i.toml

# Hardening
CapabilityBoundingSet=CAP_NET_ADMIN
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
RestrictAddressFamilies=AF_UNIX AF_NETLINK AF_INET AF_INET6
LockPersonality=yes
MemoryDenyWriteExecute=yes
RestrictNamespaces=yes
RestrictRealtime=yes
```

### Timer template

```ini
[Unit]
Description=Anycast Sentinel Timer (%i)

[Timer]
OnBootSec=30s
OnUnitActiveSec=5s
AccuracySec=1s

[Install]
WantedBy=timers.target
```

`OnBootSec` and `OnUnitActiveSec` are set by `--boot-delay` and `--interval`
at install time. The timer is re-rendered and reloaded automatically when those
values change.

To override the interval without reinstalling, use a drop-in:

```ini
# /etc/systemd/system/anycast-sentinel@myservice.timer.d/interval.conf
[Timer]
OnUnitActiveSec=10s
```

### Drop-in overrides

All hardening directives can be relaxed via drop-ins without modifying the
installed unit files. Create a file under
`/etc/systemd/system/anycast-sentinel@.service.d/` and run
`systemctl daemon-reload`.

**`MemoryDenyWriteExecute`**: Some interpreters (Python, Ruby, Node.js) use
JIT compilation or write+execute memory mappings. If a `command` check invokes
such an interpreter, the process will be killed with `SIGSYS`. Fix with:

```ini
# /etc/systemd/system/anycast-sentinel@.service.d/override.conf
[Service]
MemoryDenyWriteExecute=no
```

---

## Output

Output is line-oriented and grep-friendly. Each line is prefixed with the
subsystem that produced it.

### All checks pass, address absent → adding

```
check [systemd]: "my-service" → passed (active)
check [tcp]: "my-service-port" → passed (connected)
check [command]: "custom-check" → passed (exit 0)
health: all checks passed
route [eth0/203.0.113.10]: absent → adding
```

### Check fails, address present → removing

```
check [systemd]: "my-service" → failed (inactive)
health: checks failed
route [eth0/203.0.113.10]: present → removing
```

### All checks pass, address already present → no change

```
check [systemd]: "my-service" → passed (active)
health: all checks passed
route [eth0/203.0.113.10]: present → keeping
```

### Dry run

```
check [systemd]: "my-service" → passed (active)
health: all checks passed
route [eth0/203.0.113.10]: absent → adding (dry run)
```

---

## Operational Notes

- **Stateless**: no counters, grace windows, or state files. Each invocation
  is independent. Flap frequency is determined entirely by the timer interval.
- **Idempotent**: adding an already-present address and removing an absent
  address are both no-ops.
- **Dual-stack**: if both `ip4` and `ip6` are configured, both must be present
  for the check to pass. Both are added or removed together.
- **Capabilities**: the service requires `CAP_NET_ADMIN` for netlink address
  management. No other elevated privileges are needed under normal use.
- **Command checks**: run via `/bin/sh -c`. Standard output and stderr are
  captured; on failure, the first line of output is included in the log.
- **Systemd checks**: connect to D-Bus to read the `ActiveState` property.
  The D-Bus socket must be accessible (it is under the default service hardening).

---

## Building

```sh
# Development build (debug symbols, version = "dev" if no tags)
make build

# Run tests
make test

# Run static analysis
make vet

# Release builds for linux/amd64 and linux/arm64
make release
```

Release binaries are placed in `dist/` as
`anycast-sentinel_<version>_<os>_<arch>.tar.gz`. Version is taken from the
most recent git tag.

---

## Tests

Tests live alongside the packages they cover. Run them all with:

```sh
make test
# or
go test ./...
```

### `internal/config` — config loading (`load_test.go`)

Table-driven tests covering every validation path in `Load()`:

- Valid configurations: IPv4-only, IPv6-only, dual-stack, all three check
  types, optional timeouts
- `[general]` errors: missing `dev`, missing both IPs, invalid IP address
- `[[checks]]` errors: no checks, missing `name`, missing `type`, unknown type
- Per-type field validation: missing `unit`, `host`, `port`, `command`;
  invalid timeout strings; `timeout` on a `systemd` check (config error)
- TOML structure: unknown fields rejected by `DisallowUnknownFields()`
- Missing config file returns an error

### `internal/health` — health engine (`health_test.go`)

Unit tests for each check type. Systemd checks use a `mockQuerier` (satisfies
the `stateQuerier` interface) so no real D-Bus connection is required. TCP
checks bind a real listener on a random port. Command checks run real shell
commands.

- **systemd**: active unit passes; inactive/error/nil-client fails
- **tcp**: open port passes; closed port fails; respects `timeout`
- **command**: `true` passes; `false` fails; first line of output appears in
  the error on failure; command killed when `timeout` is exceeded
- **AllHealthy**: stops at the first failure and names it in the error;
  all-pass returns nil; unknown check type returns an error

---

## License

MIT
