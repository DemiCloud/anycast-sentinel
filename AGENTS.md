# Agent Guidance

This file documents conventions for AI coding agents working in this repository.

---

## Privacy in Documentation & Examples

Never use a real name, username, or email in examples, docs, comments, or sample
output — use invented/placeholder values instead:

- Project/application names in paths are fine (e.g. `/opt/<app>`), but any path
  that includes a personal username must use a placeholder or variable instead
  of a real one, e.g. `/home/${user}/.config/<app>/config.toml`.
- Sample hosts/IPs in the wiki, README, or config examples must use
  documentation-only values — `example.com`, `host01.example.com`, and IPs from
  the reserved documentation ranges (`192.0.2.0/24`, `198.51.100.0/24`,
  `203.0.113.0/24`), as already done for the `route`/`check` examples below.
- People: use invented names (`alice`, `Jane Doe`) for any sample owner/contact
  field.

---

## Wiki

The project wiki lives in `.wiki/` and is published to the
[GitHub wiki](https://github.com/demicloud/anycast-sentinel/wiki).
`README.md` is a brief overview; all in-depth documentation lives in the wiki.

### Wiki pages

| File | Purpose |
|------|---------|
| `.wiki/Home.md` | Hub page and navigation |
| `.wiki/Installation.md` | Build, install, and uninstall steps |
| `.wiki/CLI-Reference.md` | All subcommands, flags, exit codes, and example output |
| `.wiki/Configuration.md` | TOML config format, all fields including `http` check |
| `.wiki/Health-Checks.md` | Per-type health check behaviour and field reference |
| `.wiki/Systemd-Units.md` | Unit templates, timer tuning, and drop-in override examples |

### When to update the wiki

Update the relevant wiki page(s) whenever you:

- Add, rename, or remove a CLI subcommand or flag
- Add, change, or remove a config field or check type
- Change default values for any field or flag
- Modify the systemd unit or timer templates
- Change build, install, or uninstall steps

Also update `README.md` if the features list or subcommands table becomes stale.

---

## Local wiki setup

The `.wiki/` directory is a standalone git repository whose remote points at
the GitHub wiki (`https://github.com/demicloud/anycast-sentinel/wiki`).
It is listed in `.gitignore` and is never committed to the main repo.

To configure it locally:

1. Create the directory structure:
   ```sh
   mkdir -p .wiki/.git
   ```

2. Copy the parent repo's git config into it:
   ```sh
   cp .git/config .wiki/.git/config
   ```

3. Edit `.wiki/.git/config` and replace every occurrence of `.git` with `.wiki.git`.
   The most important change is the remote URL — it should end with `.wiki.git`:
   ```ini
   # Before:
   url = git@github.com:demicloud/anycast-sentinel.git

   # After:
   url = git@github.com:demicloud/anycast-sentinel.wiki.git
   ```
   Any `worktree` or `gitdir` paths that reference `.git` should likewise be updated
   to reference `.wiki.git`.

4. Initialize HEAD and fetch the wiki content:
   ```sh
   cd .wiki
   git init
   git remote add origin git@github.com:demicloud/anycast-sentinel.wiki.git
   git fetch origin
   git checkout -b master origin/master
   ```

After setup, commit and push wiki changes from inside `.wiki/` as a normal git
repository. The main repo is unaffected.

---

## Code conventions

- Output lines follow the pattern: `component [context]: status`
  - e.g. `check [tcp]: "my-port" → passed (connected)`
  - e.g. `route [eth0/203.0.113.10]: absent → adding`
- Config validation errors follow: `check "name": problem description`
- All public packages have table-driven tests in `*_test.go` files alongside
  the source
- New check types require changes in: `config/config.go` (constants),
  `config/load.go` (validation), `health/health.go` (engine), and both
  `config/load_test.go` and `health/health_test.go` (tests), plus
  `.wiki/Health-Checks.md` and `.wiki/Configuration.md`
