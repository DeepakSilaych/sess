<div align="center">

# sess — tmux session manager

> One tool. No worktrees. No containers. Sessions that survive everything.

```
sess new feature-auth          # create + auto-attach
...work in session...
Ctrl+b d                       # detach (session persists in tmux)
sess feature-auth              # reattach (auto-reconnects on SSH drop)

sess rm feature-auth           # destroy session
```

</div>

## What it does

- **tmux** for persistence — sessions survive SSH drops, laptop sleep, network failures; auto-reconnect loop built in
- **Native tmux status bar** — shows session name, git branch (tracks current pane dir), time
- **Connection log** — tracks how each session ended: detach, exit, or drop (SSH timeout)
- **Remote sessions** — SSH to VM, sessions live there, auto-reconnect on wake

## All commands

```
sess new <name> [branch]        Create session + auto-attach
                                --remote <name>  use a specific remote
                                --local          force local, skip remotes
sess <name>                     Attach to existing session
                                Ctrl+b d to detach

sess ls                         List sessions
sess rm <name>                  Remove session (kills tmux)

sess diff <name> [path]         Git diff in session's directory
sess log <name> [N]             Activity log (default: 20)
sess connections <name> [N]     Connection log (detach/exit/drop)
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

## Status bar

Inside every session, the tmux status bar shows:

```
 session-name                    main  rfq-modular  22:18
```

- **Left**: session name
- **Right**: git branch of current pane directory + dir name + time
- Updates every 3 seconds; tracks wherever you `cd` — not where the session was started

## Remote workflow (macOS → Linux VM)

Sessions live on the VM. Your laptop is just a terminal.

```bash
# One-time setup — provisions tmux, git, and sess itself on the VM
sess init user@dev-vm

# Daily workflow
sess new feature-auth           # SSH to VM, create session, attach
sess feature-auth               # SSH to VM, reattach
sess code feature-auth          # open Cursor via Remote-SSH
sess ssh                        # plain SSH to VM

# Close laptop. Open laptop.
# → sess auto-reconnects when SSH comes back (built-in retry loop)
```

`sess init` connects over SSH, installs `tmux`/`git` with whatever package manager
it finds (`apt`/`dnf`/`yum`/`apk`/`pacman`/`brew`), copies this `sess` script to
`~/bin/sess` on the host, puts `~/bin` on `PATH`, and registers the host as a remote
— equivalent to what `sess remote add` expects to already be true.

If the host is already provisioned, `sess remote add [name] <user@host>` just
registers it without touching anything remotely.

No sleepwatcher needed — `sess` loops and reconnects automatically when the network is back.

### Multiple remotes

```bash
sess init user@dev-vm-1 dev1
sess init user@dev-vm-2 dev2

sess new feature-auth            # more than one remote configured → prompts you to pick
sess new feature-auth --remote dev2   # skip the prompt, target dev2 directly
sess new feature-auth --local         # skip remotes entirely, create locally
```

With exactly one remote configured, `sess new` uses it automatically — no prompt.
With zero, it creates the session locally, same as always.

## Connection log

Every time a session ends, sess records how:

| Event | Meaning |
|---|---|
| `detach` | You pressed Ctrl+b d (intentional) |
| `exit` | Shell exited (intentional) |
| `drop` | SSH timeout, network failure, laptop sleep (unintentional) |

```bash
sess connections feature-auth
# 2026-07-10T14:30:00  detach (Ctrl+b d)
# 2026-07-10T15:12:00  drop
# 2026-07-10T16:45:00  exit
```

## Platform support

| Feature | macOS | Linux |
|---|---|---|
| tmux sessions | ✅ | ✅ |
| Status bar | ✅ | ✅ |
| Auto-reconnect on SSH drop | ✅ | ✅ |
| Connection log | ✅ | ✅ |
| Remote (SSH to VM) | ✅ | ✅ |
| `sess code` | ✅ (Cursor/VS Code) | ✅ |

## Architecture

```
~/.sess/sessions/<name>/
├── tmux-init.sh        ← sets SESS_* env vars, execs shell
├── state               ← session metadata (branch, cwd)
├── log                 ← activity log
└── connections         ← connection log (detach/exit/drop)
```

## Install

### From npm

```bash
npx sess-sh          # try it without installing
npm install -g sess-sh
```

### From source

```bash
git clone https://github.com/deepaksilaych/sess.git
cd sess
sudo make install
```

Or just use locally:

```bash
chmod +x bin/sess
export PATH="$PWD/bin:$PATH"
```

### Prerequisites

- **tmux** — session persistence (`apt install tmux` / `brew install tmux`)
- **git** — version control
- **bash** — shell
- **SSH** — for remote sessions

## Comparison

| | sess | git worktree | tmux | Docker | VM per agent |
|---|---|---|---|---|---|
| Creation time | ~instant | ~1s | N/A | ~seconds | ~minutes |
| SSH persistence | ✅ auto-reconnect | ❌ | ✅ | ⚠️ manual | ✅ |
| Path stability | ✅ same path | ❌ varies | N/A | ⚠️ mapping | ⚠️ mapping |
| Agent-friendly | ✅ | ❌ | ❌ | ⚠️ | ⚠️ |
| Status bar | ✅ tmux native | ❌ | ✅ | ❌ | ❌ |

## License

MIT
