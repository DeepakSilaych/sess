# sess — dtach sessions with overlay isolation

> One tool. No worktrees. No containers. No tmux.

```
sess new feature-auth          # create + auto-attach
...work in session...
Ctrl+\                         # detach (session persists)
sess feature-auth              # reattach

sess overlay new feature-auth bugfix-1222 main   # create overlay
sess overlay switch feature-auth bugfix-1222     # switch overlay
sess rm feature-auth           # destroy session + overlays
```

## What it does

- **dtach** for persistence — single Unix socket per session, no server process, Ctrl+\ to detach
- **OverlayFS** for filesystem isolation (Linux) — N independent writable views of the same repo, only deltas stored
- **APFS clones** on macOS — copy-on-write clones as overlay fallback
- **PROMPT_COMMAND status bar** — plain text, no colors: `session | overlay | branch | ~/path     HH:MM`
- **Connection log** — tracks how each session ended: detach, exit, or drop (SSH timeout)
- **Remote sessions** — SSH to VM, sessions live there, auto-reconnect on wake

## All commands

```
sess new <name> [branch]        Create session + auto-attach
sess <name>                     Attach to existing session
                                Ctrl+\ to detach

sess ls                         List sessions
sess rm <name>                  Remove session (unmounts overlays)

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

## Remote workflow (macOS → Linux VM)

Sessions live on the VM. Your laptop is just a terminal.

```
# One-time setup
sess remote add default user@dev-vm

# Daily workflow
sess new feature-auth           # SSH to VM, create session, attach
sess feature-auth               # SSH to VM, reattach
sess code feature-auth          # open Cursor via Remote-SSH
sess ssh                        # plain SSH to VM

# Close laptop. Open laptop.
sess up                         # reconnects to all active sessions
```

For fully automatic reconnect on wake, add to `~/.wakeup`:
```bash
sleep 3 && sess up &
```

Install SleepWatcher: `brew install sleepwatcher`

## Connection log

Every time a session ends, sess records how:

| Event | Meaning |
|---|---|
| `detach` | You pressed Ctrl+\ (intentional) |
| `exit` | You typed `exit` or Ctrl-D (intentional) |
| `drop` | SSH timeout, network failure, laptop sleep (unintentional) |

```bash
sess connections feature-auth
# 2026-07-09T14:30:00  detach (Ctrl+\)
# 2026-07-09T15:12:00  drop (exit code 141)
# 2026-07-09T16:45:00  exit
```

## Platform support

| Feature | macOS | Linux |
|---|---|---|
| dtach sessions | ✅ | ✅ |
| Status bar | ✅ | ✅ |
| Connection log | ✅ | ✅ |
| Remote (SSH to VM) | ✅ | ✅ |
| Auto-reconnect on wake | ✅ (SleepWatcher) | ✅ (systemd) |
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
├── socket              ← dtach Unix socket
├── hook.sh             ← PROMPT_COMMAND status bar hook
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

### Via npx

```bash
npx sess-cli
```

This installs the `sess` command globally.

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

- **dtach** — session persistence (`apt install dtach` / `brew install dtach`)
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
| SSH persistence | ✅ dtach | ❌ | ✅ | ⚠️ manual | ✅ |
| Path stability | ✅ same path | ❌ varies | N/A | ⚠️ mapping | ⚠️ mapping |
| Agent-friendly | ✅ | ❌ | ❌ | ⚠️ | ⚠️ |
| No extra process | ✅ | ✅ | ❌ (server) | ❌ (daemon) | ❌ (VM) |

## License

MIT