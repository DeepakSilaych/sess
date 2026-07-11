# sess — tmux sessions with overlay isolation

> One tool. No worktrees. No containers. Sessions that survive everything.

```
sess new feature-auth          # create + auto-attach
...work in session...
Ctrl+b d                       # detach (session persists in tmux)
sess feature-auth              # reattach (auto-reconnects on SSH drop)

sess overlay new feature-auth bugfix-1222 main   # create overlay
sess overlay switch feature-auth bugfix-1222     # switch overlay
sess rm feature-auth           # destroy session + overlays
```

## What it does

- **tmux** for persistence — sessions survive SSH drops, laptop sleep, network failures; auto-reconnect loop built in
- **Native tmux status bar** — shows session name, overlay, git branch (tracks current pane dir), time
- **OverlayFS** for filesystem isolation (Linux) — N independent writable views of the same repo, only deltas stored
- **APFS clones** on macOS — copy-on-write clones as overlay fallback
- **Connection log** — tracks how each session ended: detach, exit, or drop (SSH timeout)
- **Remote sessions** — SSH to VM, sessions live there, auto-reconnect on wake

## All commands

```
sess new <name> [branch]        Create session + auto-attach
sess <name>                     Attach to existing session
                                Ctrl+b d to detach

sess ls                         List sessions
sess rm <name>                  Remove session (kills tmux, unmounts overlays)

sess overlay new <s> <ov> [br]  Create overlay (from latest default branch)
sess overlay switch <s> <ov>    Switch active overlay
sess overlay ls <s>             List overlays
sess overlay rm <s> <ov>        Remove overlay

sess diff <name> [path]         Git diff in session's overlay
sess log <name> [N]             Activity log (default: 20)
sess connections <name> [N]     Connection log (detach/exit/drop)
sess path <name>                Print overlay/cwd path (for scripts)
sess code <name>                Open Cursor/VS Code for session
sess status [name]              Session info or overall status

sess ssh [args]                 SSH to configured remote VM
sess up                         Reconnect to previously active sessions
sess remote add [name] <host>   Configure a remote VM
sess remote ls                  List configured remotes
sess remote rm [name]           Remove a remote

sess doctor                     Check prerequisites
sess help                       Show help
sess version                    Show version
```

## Status bar

Inside every session, the tmux status bar shows:

```
 session-name  ⬡ overlay-name          main  rfq-modular  22:18
```

- **Left**: session name + active overlay (if any)
- **Right**: git branch of current pane directory + dir name + time
- Updates every 3 seconds; tracks wherever you `cd` — not where the session was started

## Remote workflow (macOS → Linux VM)

Sessions live on the VM. Your laptop is just a terminal.

```bash
# One-time setup
sess remote add default user@dev-vm

# Daily workflow
sess new feature-auth           # SSH to VM, create session, attach
sess feature-auth               # SSH to VM, reattach
sess code feature-auth          # open Cursor via Remote-SSH
sess ssh                        # plain SSH to VM

# Close laptop. Open laptop.
# → sess auto-reconnects when SSH comes back (built-in retry loop)
```

No sleepwatcher needed — `sess` loops and reconnects automatically when the network is back.

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
| Overlay isolation | ✅ APFS clone | ✅ OverlayFS mount |

## Overlay isolation

### Linux (OverlayFS)

```bash
sess overlay new feature-auth bugfix-1222 main
```

Creates an OverlayFS mount — instant, only deltas stored. Writing to the overlay does NOT affect the original repo.

### macOS (APFS clones)

Same command, different kernel primitive. Uses `cp -cR` for copy-on-write clones on APFS. Same experience — isolated writes, shared unchanged files.

## Architecture

```
~/.sess/sessions/<name>/
├── tmux-init.sh        ← sets SESS_* env vars, execs shell
├── state               ← session metadata (branch, overlay, cwd)
├── log                 ← activity log
├── connections         ← connection log (detach/exit/drop)
└── overlays/
    └── bugfix-1222/
        ├── upper/      ← OverlayFS writable layer (deltas only)
        ├── work/       ← OverlayFS kernel work dir
        └── merged/     ← unified mount point (what you see)
```

## Install

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
- **OverlayFS** — filesystem isolation (Linux only, standard on modern kernels)
- **SSH** — for remote sessions

## Comparison

| | sess | git worktree | tmux | Docker | VM per agent |
|---|---|---|---|---|---|
| Isolation | ✅ per-overlay | ⚠️ branch-level | ❌ shared FS | ✅ full | ✅ full |
| Creation time | ~instant | ~1s | N/A | ~seconds | ~minutes |
| Disk per session | Delta only | Full copy | 0 | Full copy | Full VM |
| SSH persistence | ✅ auto-reconnect | ❌ | ✅ | ⚠️ manual | ✅ |
| Path stability | ✅ same path | ❌ varies | N/A | ⚠️ mapping | ⚠️ mapping |
| Agent-friendly | ✅ | ❌ | ❌ | ⚠️ | ⚠️ |
| Status bar | ✅ tmux native | ❌ | ✅ | ❌ | ❌ |

## License

MIT
