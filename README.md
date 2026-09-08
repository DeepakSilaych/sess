<div align="center">

# sess

tmux sessions that remember where they were and reconnect over SSH when the link drops.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> One tool. No worktrees. No containers.

</div>

## Why

Close the laptop and your SSH session dies. tmux keeps the shell alive on the server, but getting back means typing `ssh`, then `tmux attach`, then remembering which session was in which directory. mosh and Eternal Terminal fix the transport, but each needs its own server daemon on every box.

`sess` is one bash script on top of `ssh` and `tmux`. It keeps a small state dir per session (cwd, branch, logs), copies itself to your VM with `sess init`, and when you attach to a remote session it re-runs `ssh -t host "sess <name>"` every 3 seconds until you stop it. Nothing else to install, nothing listening on a port.

## Demo

```
sess new feature-auth          # create + auto-attach (cwd = where you ran it)
...work in session...
Ctrl+b d                       # detach (session persists in tmux)
sess feature-auth              # reattach (loops on SSH drop if a remote is set)

sess rm feature-auth           # destroy session
```

Inside a session the tmux status bar shows the session name on the left and, on the right, the git branch of the current pane directory, the directory name, and the time. It refreshes every 3 seconds and follows wherever you `cd`.

```
 session-name                    main  rfq-modular  22:18
```

<!-- TODO: gif of a remote session reconnecting after the network drops -->

## Quickstart

Prerequisites: `tmux`, `git`, `bash`, and `ssh` for remote sessions. `sess doctor` checks all four. Runs on macOS and Linux (`package.json` `os`).

Install from npm:

```bash
npx sess-sh                # run once without installing
npm install -g sess-sh
```

Or from source (installs `bin/sess` plus bash and zsh completions under `PREFIX`, default `/usr/local`):

```bash
git clone https://github.com/deepaksilaych/sess.git
cd sess
sudo make install          # or: PREFIX=$HOME/.local make install
```

Or just put `bin/` on your PATH:

```bash
chmod +x bin/sess
export PATH="$PWD/bin:$PATH"
```

Local sessions:

```bash
cd ~/code/my-repo
sess new feature-auth      # creates ~/.sess/sessions/feature-auth, starts tmux, attaches
# Ctrl+b d to detach
sess ls                    # SESSION / BRANCH / STATUS / CREATED
sess feature-auth          # reattach
```

Remote sessions (sessions live on the VM, the laptop is just a terminal):

```bash
sess init user@dev-vm      # one time: installs tmux+git, copies sess to ~/bin/sess, registers "default"
sess new feature-auth      # runs "sess new feature-auth --local" on the VM over ssh -t, attaches
sess feature-auth          # ssh -t + attach, retried every 3s after any disconnect
sess ssh                   # plain ssh to the VM
```

Verify:

```bash
sess status                # version, state dir, session count, previously active sessions
sess connections feature-auth
```

## How it works

Everything is in `bin/sess` (one bash script, ~970 lines). The main `case` at the bottom dispatches subcommands; any other word is treated as a session name and goes to `cmd_attach`.

```
laptop                                              dev VM (same script at ~/bin/sess)
------                                              ----------------------------------
sess feature-auth
  |
  | ~/.sess/remote has default.host?
  |
  no ---> cmd_attach (local)
  |        _tmux_start: tmux has-session? else new-session -d -c $SESS_CWD tmux-init.sh
  |        _apply_tmux_status; tmux attach-session
  |        on return: connections += detach | exit
  |
  yes --> cmd_remote_attach
           while true:
             ssh -t user@vm "sess feature-auth"  ----->  cmd_attach (local, on the VM)
                     ^                                     tmux attach ... returns on
                     |  ssh exits (drop / Ctrl+b d / exit)  detach, exit, or SIGHUP
                     |
             print "[sess] disconnected. reconnecting in 3s... (Ctrl+C to stop)"
             sleep 3
```

1. `sess new <name>` writes `~/.sess/sessions/<name>/state` (`SESS_SESSION`, `SESS_BRANCH`, `SESS_CWD=$(pwd)`, `SESS_CREATED`), appends to `log`, adds the name to `~/.sess/active-sessions`, then calls `cmd_attach`. If a remote is selected it instead `exec`s `ssh -t host "sess new <name> --local"`, so the state lives on the VM.
2. `cmd_attach` sources `state`, and `_tmux_start` creates the tmux session on first use with `tmux new-session -d -s <name> -c <cwd> tmux-init.sh`. That init script exports `SESS_SESSION`, `SESS_BRANCH`, `SESS_DIR`, `cd`s to the saved cwd and `exec`s `$SHELL`. If the tmux session already exists it is reused.
3. `_apply_tmux_status` runs on every attach and sets session-scoped tmux options: `status-interval 3`, session name on the left, `git rev-parse --abbrev-ref HEAD` in `#{pane_current_path}` plus `#{b:pane_current_path}` and `%H:%M` on the right. No `.tmux.conf` changes.
4. When `tmux attach-session` returns, `cmd_attach` checks `tmux has-session`. Still there means you detached (`detach` is written to `connections`); gone means the shell exited (`exit`, and the name is dropped from `active-sessions`).
5. The reconnect loop is `cmd_remote_attach`. It only runs when `~/.sess/remote` has a `default.host=` line. It does not look at the ssh exit code: any return, including an intentional `Ctrl+b d`, prints the disconnected message, sleeps 3 seconds and runs ssh again. `Ctrl+C` during that 3 second wait ends the loop. No `ServerAliveInterval` is set, so how fast a dead link is noticed depends on your ssh config.
6. `sess init` (`cmd_init`) installs tmux and git with whichever of `apt-get`, `dnf`, `yum`, `apk`, `pacman`, `brew` it finds, `scp`s the running script (symlinks resolved by `_self_path`) to `~/bin/sess`, appends `~/bin` to PATH in `.bashrc`, `.zshrc` and `.profile`, checks `~/bin/sess version`, and writes `name.host=user@host` to `~/.sess/remote`.
7. `sess up` (`cmd_up`) reads `~/.sess/active-sessions`. Locally it reattaches the last one, or if you are already inside tmux it starts all of them and `switch-client`s to the first. With a default remote it checks each name with `ssh host "test -d ~/.sess/sessions/<name>"`, then on macOS opens one Terminal.app tab per session via `osascript`, and on Linux runs `ssh -t host "sess <name>"` one after another.

### Compared with

| | ssh + tmux by hand | mosh | Eternal Terminal | sess |
|---|---|---|---|---|
| Server side | tmux | mosh-server | etserver | tmux + `~/bin/sess` (copied by `sess init`) |
| Transport | ssh | UDP, own protocol | own TCP protocol | ssh |
| After a drop | you re-run `ssh` and `tmux attach` | link resumes by itself | link resumes by itself | `sess` re-runs `ssh -t host "sess <name>"` every 3s |
| Remembers cwd, branch and how each attach ended | no | no | no | yes, `~/.sess/sessions/<name>/` |

## Features

| Feature | Where |
|---|---|
| Per-session state dir: `state`, `log`, `connections`, `tmux-init.sh` | `cmd_new`, `_tmux_start`, `_log_event`, `_conn_log` |
| `SESS_SESSION`, `SESS_BRANCH`, `SESS_DIR` exported inside every session shell | `tmux-init.sh` written by `_tmux_start` |
| tmux status bar: session name, branch of current pane dir, dir name, HH:MM, 3s refresh | `_apply_tmux_status` |
| Connection log with `detach` and `exit` events | `cmd_attach`, `cmd_connections` |
| Activity log: created, attached, detached, exited, removed, diff, code | `_log_event`, `cmd_log` |
| Remote reconnect loop, 3s retry, stop with Ctrl+C | `cmd_remote_attach` |
| `sess init` provisions a host: package install, copy script, PATH, register | `cmd_init` |
| Multiple remotes: `--remote <name>`, `--local`, interactive pick, `default` first when non-interactive | `_select_remote` |
| `sess up` reattaches previously active sessions; macOS opens a Terminal tab per session | `cmd_up`, `_local_up`, `_remote_up` |
| `sess code` opens `$SESS_EDITOR`, else `cursor`, else `code`; remote via `--remote host` then a `vscode-remote://` URI | `cmd_code` |
| bash and zsh completion for commands, session names, git branches, remote names | `etc/bash-completion/sess`, `etc/zsh-completion/_sess` |
| npm wrapper so `npx sess-sh` works; zero npm dependencies | `bin/sess-cli.js`, `package.json` |

All commands (`sess help`):

```
sess new <name> [branch]        Create session + auto-attach
                                --remote <name>  use a specific remote
                                --local          force local, skip remotes
sess <name>                     Attach to existing session (also: sess attach <name>)
                                Ctrl+b d to detach

sess ls                         List sessions
sess rm <name>                  Remove session (kills tmux)

sess diff <name> [path]         Git diff in session's directory
sess log <name> [N]             Activity log (default: 20)
sess connections <name> [N]     Connection log (detach/exit)
sess path <name>                Print session's cwd (for scripts)
sess code <name>                Open Cursor/VS Code for session
sess status [name]              Session info or overall status

sess ssh [args]                 SSH to configured remote VM
sess up                         Reconnect to previously active sessions
sess init <user@host> [name]    Provision a host (tmux+git+sess) + register it
sess remote add [name] <host>   Register an already-provisioned remote
sess remote ls                  List configured remotes
sess remote rm [name]           Remove a remote

sess doctor                     Check prerequisites
sess help                       Show help
sess version                    Show version
```

Aliases: `ls`/`list`, `rm`/`remove`, `connections`/`conn`. `[branch]` is a label stored in `state` and exported as `SESS_BRANCH`; it defaults to the current branch of the cwd (else `main`) and is never checked out.

Which commands go to the VM when a `default` remote is configured:

| Proxied over ssh | Run against the laptop's `~/.sess` only |
|---|---|
| `sess new` (unless `--local`), `sess <name>`, `sess ls`, `sess rm`, `sess up`, `sess ssh` | `sess diff`, `sess log`, `sess connections`, `sess path`, `sess status <name>`, `sess code` |

## Configuration

Environment variables read by `bin/sess`:

| Variable | Default | Purpose |
|---|---|---|
| `SESS_DIR` | `~/.sess` | State directory (sessions, remotes, active list) |
| `SESS_EDITOR` | `cursor`, then `code` | Editor command for `sess code` |
| `SHELL` | `bash` | Shell exec'd inside a new tmux session |

Set inside every session shell by `tmux-init.sh`: `SESS_SESSION`, `SESS_BRANCH`, `SESS_DIR`.

Files under `SESS_DIR`:

| Path | Contents |
|---|---|
| `remote` | `name.host=user@host` lines; `default` is the one attach/ls/rm/up/ssh use |
| `active-sessions` | One session name per line, most recent last |
| `sessions/<name>/state` | `SESS_SESSION`, `SESS_BRANCH`, `SESS_CWD`, `SESS_CREATED` (sourced by the script) |
| `sessions/<name>/log` | Timestamped activity log |
| `sessions/<name>/connections` | Timestamped `detach` / `exit` events |
| `sessions/<name>/tmux-init.sh` | Generated init script for the tmux session |

Makefile: `PREFIX` (default `/usr/local`) and `DESTDIR` control where `make install` puts the script and completions.

Optional macOS hook: `etc/wakeup` is a SleepWatcher script that runs `sess up` in the background 3 seconds after wake. Install with `brew install sleepwatcher`, `cp etc/wakeup ~/.wakeup`, `chmod +x ~/.wakeup`. An already-attached remote session reconnects on its own through the retry loop; the hook is only for reopening sessions you were not attached to.

## Design decisions

- One script, copied as-is. `sess init` scps `bin/sess` to the VM, so every remote command is just `ssh host "sess ..."` running the same code. No daemon, no port, no protocol of its own.
- Reconnect is a blind retry. `cmd_remote_attach` ignores the ssh exit status and reattaches after 3 seconds no matter why ssh returned. Simple and hard to break, but it means `Ctrl+b d` on a remote session comes back after 3 seconds; `Ctrl+C` during the wait is the way out.
- State is plain text. `state` is a shell file that gets `source`d, `remote` is `key=value`, logs are one line per event. Easy to `cat`, `grep` and `scp`, and easy to hand-edit if something goes wrong.
- tmux options are set per session with `tmux set-option -t <name>`, applied on every attach. Your `.tmux.conf` is untouched, but the status line of a sess session is always sess's.
- The branch argument is metadata, not a checkout. Sessions are meant to be cheap labels over one working tree, which is the "no worktrees" part of the tagline.
- Only the remote named `default` drives attach, ls, rm, up and ssh. Other remote names exist for `sess new --remote <name>` and the interactive picker.

## Project layout

```
bin/sess                    The tool. Single bash script, VERSION at the top.
bin/sess-cli.js             npm shim: execFileSync(bin/sess, argv)
etc/bash-completion/sess    bash completion
etc/zsh-completion/_sess    zsh completion
etc/wakeup                  Optional macOS SleepWatcher hook that runs `sess up`
docs/index.html             Static landing page
test/test_sess.sh           Smoke test (uses a temp SESS_DIR, needs git)
Makefile                    install / uninstall / test / clean
package.json                npm package `sess-sh`, bin `sess`
```

## Development

```bash
make test                  # bash -n on bin/sess, loads the bash completion
bash test/test_sess.sh     # smoke test: version, help, doctor, ls, status, path, log, rm
```

`make test` does not run `test/test_sess.sh`; run both. The smoke test creates a throwaway git repo and a session state dir under `/tmp` and cleans them up on exit. It never attaches (that needs a TTY).

There is no linter or CI workflow in the repo. The version string lives in two places, `VERSION=` in `bin/sess` and `"version"` in `package.json`; bump both before `npm publish`. The published files are the `files` list in `package.json`.

## Limitations

Things you can see in the code today:

- No `drop` event. The connection log only ever gets `detach` or `exit`; the help text and `docs/index.html` still mention `drop`. A network drop usually kills the remote script with the SSH session, so nothing is written.
- `sess up` with a default remote reads the laptop's `~/.sess/active-sessions`, but the remote paths of `sess new` and `sess <name>` hand off to ssh before writing it. In a pure remote workflow it reports "No previously active sessions".
- The Terminal tabs that `sess up` opens on macOS run a plain `ssh -t host 'sess <name>'`, without the retry loop. On Linux the remote `sess up` attaches one session at a time.
- `sess diff`, `log`, `connections`, `path`, `status <name>` and `code` only read the laptop's state dir, so with a default remote they fail for sessions that exist only on the VM. Run them on the VM through `sess ssh` instead.
- A session created with `sess new --remote dev2` can only be reattached with `sess <name>` if `dev2` is also the `default` remote.

## Contributing

Open an issue or PR at [deepaksilaych/sess](https://github.com/deepaksilaych/sess). Run `make test` and `bash test/test_sess.sh` before pushing.

## License

MIT. See [LICENSE](LICENSE).
