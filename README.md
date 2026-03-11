# anycast-sentinel

anycast-sentinel is a stateless, one‑shot health‑gated Anycast announcer. It evaluates a list of health checks, prints structured results, and adds or removes an Anycast IP address on an interface using netlink. Systemd timers control execution frequency; the sentinel itself never loops.

All checks must pass (logical AND). Any failure withdraws the Anycast address immediately.

---

## Features

- Stateless one‑shot execution
- Multiple health‑check types:
  - systemd unit state
  - TCP connect
  - command execution
- AND‑semantics: all checks must pass or the route is withdrawn
- IPv4 `/32` and IPv6 `/128` advertisement via netlink
- Deterministic, grep‑friendly output
- Systemd templated units with dynamic binary path injection
- Safe template updates: overwrite only when content differs
- Operator overrides supported via systemd drop‑ins

---

## Installation

Install systemd templates and enable an instance:

`anycast-sentinel --install <instance>`

This performs:

1. Resolve the real binary path (`ExecStart={{.ExecPath}}`)
2. Render and install:
   - `/etc/systemd/system/anycast-sentinel@.service`
   - `/etc/systemd/system/anycast-sentinel@.timer`
3. Overwrite templates only if content differs
4. Create `/etc/anycast/<instance>.toml` if missing
5. Reload systemd
6. Enable and start `anycast-sentinel@<instance>.timer`

Instances are created by enabling timers; no per‑instance unit files are written.

---

## Systemd Units

### Service Template

`[Unit]
Description=Anycast Sentinel (%i)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart={{.ExecPath}} --config /etc/anycast/%i.toml
`

### Timer Template

`[Unit]
Description=Anycast Sentinel Timer (%i)

[Timer]
OnBootSec=30s
OnUnitActiveSec=5s
AccuracySec=1s

[Install]
WantedBy=timers.target
`

---

## Configuration

Configuration files live in `/etc/anycast/<instance>.toml`.

### Example

`[general]
dev = "eth0"
ip4 = "203.0.113.10"
interval = "5s"

[[checks]]
name = "dns-service"
type = "systemd"
unit = "technitium.service"

[[checks]]
name = "dns-tcp"
type = "tcp"
host = "127.0.0.1"
port = 53
timeout_ms = 500

[[checks]]
name = "custom-script"
type = "command"
command = "/usr/local/bin/check-dns.sh"
`

### Fields

- **general.dev** — interface to bind the Anycast address to  
- **general.ip4 / general.ip6** — advertised addresses  
- **general.interval** — validated but unused (timers control frequency)  
- **checks** — list of health checks; all must pass  

### Health Check Types

- **systemd**
  - requires `unit`
- **tcp**
  - requires `host`, `port`, optional `timeout_ms`
- **command**
  - requires `command`

---

## Runtime Behavior

Each invocation:

1. Runs all health checks
2. Prints structured results:
   - `health: <name> → passed/failed (<detail>)`
3. If all pass:
   - Ensure Anycast address is present  
   - `route: absent → adding` or `route: present → keeping`
4. If any fail:
   - Ensure Anycast address is removed  
   - `route: present → removing` or `route: absent → nothing to do`

The sentinel exits immediately after making the routing decision.

---

## Logging Output Examples

### All checks pass

`health: systemd technitium.service → passed (active)
health: tcp dns-tcp → passed (connected)
health: command custom-script → passed (exit 0)
health: all checks passed
route: absent → adding
`

### Failure

`health: systemd technitium.service → failed (inactive)
health: failure detected
route: present → removing
`

---

## Operational Model

- The sentinel is stateless; no counters or grace windows.
- Systemd timers determine frequency.
- Operators modify behavior via:
  - `/etc/anycast/<instance>.toml`
  - `/etc/systemd/system/anycast-sentinel@.service.d/*.conf`
- Template unit files are owned by the sentinel and overwritten when changed.

---

## Building

`make build`

Produces a static binary with version metadata injected via `-ldflags`.

---

## License

MIT (or your preferred license)

