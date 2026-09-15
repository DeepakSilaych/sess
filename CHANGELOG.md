# Changelog

## Unreleased

- Homebrew installation through `deepaksilaych/tap/sess`, including shell completions.
- `sess-cli` npm launcher for `npx sess-cli` and global installation, with verified platform downloads and a local cache.

## [0.6.0](https://github.com/DeepakSilaych/sess/releases/tag/v0.6.0) — 2026-09-15

The first tagged release of the zmx-based sess: a persistent SSH terminal with a Go CLI and interactive session browser.

### Added

- `sess init <host>` installs a remote helper and SHA-256-verified zmx 0.8.1 for the remote account.
- `sess set --host <host>` saves a default SSH destination. Session commands consistently accept `--host` / `-h` overrides.
- `new` / `n`, `attach` / `a`, `remove` / `rm`, `ls`, `detach`, `doctor`, and shell completion.
- A searchable TUI with host switching, asynchronous refresh, create/attach/remove actions, and return to the browser after detach.
- `new --detach`, `ls --json`, and `ls --quiet` for scripting.
- A versioned SSH command protocol, random session identities, private backend runtime directories, and atomic local configuration.
- macOS and Linux archives for ARM64 and AMD64, with all four remote-helper targets embedded in each client.
- Local unit, race, native-backend, and Docker SSH integration checks.

### Changed

- zmx replaces tmux as the session backend. The terminal application owns windows and tabs.
- Go replaces the single Bash implementation. Node.js and npm packaging are removed.
- Explicit detach and normal shell exit end the attachment. Only classified transient SSH failures trigger retry.
- Attaching requires an existing session. Reconnection checks its identity to detect a name reused for different work.
- Sessions are remote-only. The host is selected from the command flag or saved default; there is no local fallback.

### Removed

- tmux-specific status bars and keybindings.
- The positional branch label and the old `--remote` / `--local` routing model.
- The old `up`, `remote`, `ssh`, `diff`, `log`, `connections`, `path`, `code`, and `status` commands.
- The macOS wake-up hook and GitHub Actions workflows. Build, test, and release commands are run manually.

### Migration from 0.5

Existing tmux sessions and `~/.sess` data remain untouched. Use tmux to finish those sessions; running processes cannot be converted into zmx sessions. Install the new client, run `sess init <host>`, set your default host, and create fresh sessions.

Session operations use OpenSSH batch mode. Key authentication must work without a prompt, including unlocking encrypted keys in ssh-agent. Live sessions do not survive VM reboot, backend loss, or operating-system process cleanup.

### Validation

Race-enabled Go tests and `go vet` passed on macOS ARM64 and Linux ARM64. Native zmx checks passed on macOS. Real OpenSSH/zmx tests on a Linux ARM64 Docker VM covered provisioning, detach/reattach, persistence, forced disconnect/reconnect, multiple clients, removal, TUI operations, and isolation from ordinary zmx sessions. AMD64 targets were cross-built; they have not received the same runtime coverage.

## 0.5.0 — untagged history

The earlier Bash implementation wrapped tmux over SSH, saved state under `~/.sess`, and provided remote provisioning and a simple retry loop. See the [last pre-zmx source snapshot](https://github.com/DeepakSilaych/sess/tree/7b0e8437e6e7e93f1bba787d728f0f4a7d9c071e).
