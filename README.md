# anycast-sentinel

A stateless, one-shot health-gated anycast announcer. On each invocation it
evaluates a list of health checks and either adds or removes an anycast IP
address on a network interface using netlink. Systemd timers control execution
frequency; the sentinel itself never loops.

All checks must pass (AND semantics). Any single failure immediately withdraws
the address.

**Full documentation is in the [project wiki](https://github.com/demicloud/anycast-sentinel/wiki).**

---

## Features

- Stateless one-shot execution — no counters, no state files
- Four health check types: `systemd` unit state, `tcp` connect, `http` response, shell `command`
- AND semantics: all checks must pass or the address is withdrawn
- IPv4 `/32` and IPv6 `/128` support via netlink (dual-stack or single-stack)
- Dry-run mode: evaluates checks and reports decisions without touching routes
- `validate` subcommand: checks config validity without root or network access
- `status` subcommand: reports current address presence on the interface
- `install` / `uninstall` subcommands manage hardened systemd template units
- Operator overrides supported via systemd drop-ins
- Deterministic, grep-friendly output

---

## Quick start

```sh
make build
sudo make install
sudo anycast-sentinel install myservice

# validate config before relying on it
anycast-sentinel validate --config /etc/anycast/myservice.toml

# check current address state at any time
sudo anycast-sentinel status --config /etc/anycast/myservice.toml
```

Edit `/etc/anycast/myservice.toml` — the timer handles the rest.

---

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `run` | Evaluate health checks and manage the anycast address |
| `validate` | Check a config file for errors (no root needed) |
| `status` | Report whether configured addresses are present on the interface |
| `install` | Install systemd templates and enable a timer instance |
| `uninstall` | Disable a timer instance and remove the template unit files |
| `version` | Show version information |

See the [CLI Reference](https://github.com/demicloud/anycast-sentinel/wiki/CLI-Reference) for full flag documentation.

---

## Building

```sh
make build    # dev build → build/anycast-sentinel
make test     # run all tests
make vet      # static analysis
make release  # stripped multi-arch tarballs in dist/
```

---

## License

MIT
