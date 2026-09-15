# sess

**A persistent SSH terminal. Powered by zmx.**

Create a named terminal on your VM, work in it, and return to the same running shell later. Your commands and processes run on the VM. Your laptop provides the keyboard and display.

> **Version:** This guide covers sess **0.6.0**, powered by zmx **0.8.1**. See the [README](../README.md#install) for building and installing. Existing 0.5 tmux sessions remain separate.

## Quick start

Run these commands on your laptop after installing sess:

```bash
# Prepare the VM for persistent sessions.
sess init user@host

# Choose the host used by subsequent commands.
sess set --host user@host

# Create a session and connect to it.
sess new work

# You are now on the VM. Work normally.
cd ~/projects/my-app
npm run dev
```

Press **Ctrl+\** to detach. The shell and development server continue running on the VM.

Later, from your laptop:

```bash
sess ls
sess attach work
```

You return to the same terminal.

Use standard SSH destinations: **`user@host`**, a hostname, or an alias from your SSH configuration. `user:host` is not the username-and-host syntax used by SSH. [SSH destination syntax](https://man.openbsd.org/ssh#SYNOPSIS).

## 1. Prepare a VM

```bash
sess init user@host
```

Or use an existing SSH alias:

```bash
sess init dev
```

Run `init` on your laptop. It connects to the VM and prepares the remote side:

1. Checks that SSH access works.
2. Detects the VM's operating system and architecture.
3. Installs the compatible sess helper and zmx for your remote account, or verifies an existing installation.
4. Checks that sess can run remote session operations.
5. Reports that the host is ready.

You need an existing VM and working SSH access. Session commands and automatic reconnect use SSH batch mode: configure key authentication and unlock encrypted keys in ssh-agent before connecting. `init` does not create the VM or configure your cloud account. Its normal installation uses your remote account's writable directories. sess installs its remote helper and pinned zmx under `~/.local/share/sess/bin`. It leaves shell startup files unchanged.

**`init` does not change your default host.** Setup and host selection are separate operations. Running it again checks the installation; it refuses to replace a different managed zmx version automatically.

### SSH aliases

An alias keeps hostnames, ports, and key paths out of everyday commands. For example, in your laptop's `~/.ssh/config`:

```sshconfig
Host dev
    HostName 203.0.113.10
    User deepak
    Port 22
    IdentityFile ~/.ssh/id_ed25519
```

Replace the example address and key path with your own. First verify:

```bash
ssh dev
```

Then use `dev` anywhere sess expects a host. sess uses your system SSH client and its configured authentication and connection settings. [SSH configuration](https://man.openbsd.org/ssh_config).

## 2. Set the default host

```bash
sess set --host dev
```

Short form:

```bash
sess set -h dev
```

This saves the default on your laptop. It does not install software, start a session, or move existing sessions.

Without an override, session commands use this host:

```bash
sess new api
sess attach api
sess remove api
sess ls
```

### Override the host for one command

`new`, `attach`, `remove`, and `ls` accept `--host` or `-h`. The browser and `doctor` accept them too:

```bash
sess new tests --host staging
sess attach tests -h staging
sess ls -h staging
sess remove tests -h staging
```

An override applies only to that command. Your default remains unchanged.

The selection rule is always:

1. Use the command's `--host` / `-h`, if supplied.
2. Otherwise, use the saved default host.
3. If neither exists, explain how to set or provide a host and stop.

sess does not guess a destination or fall back to a local session. Human-readable output identifies the selected host. JSON output includes it in the `host` field; `ls --quiet` prints names only.

`-h` means **host** throughout sess. Use `--help` for help.

## 3. Create a session

```bash
sess new <name>
sess n <name>
```

Examples:

```bash
sess new api
sess n tests
sess n experiments -h staging
```

For scripts, use `sess new <name> --detach` (or `-d`) to create without attaching.

Creation starts a new persistent shell on the selected VM and immediately attaches your terminal to it. The shell starts in the remote account's home directory. Use `cd` normally after connecting.

Choose a name of 1–48 characters beginning with a letter or number, followed by letters, numbers, hyphens, or underscores: `api`, `build-42`, or `release_tests`.

If that name already exists on the selected account and host, `new` reports it and suggests `sess attach <name>`. It does not overwrite the session.

Names belong to the remote account on the VM. `api` on `dev` and `api` on `staging` are independent. Two SSH aliases reaching the same account on the same VM see the same sessions.

Creating another session does not create another repository checkout. Sessions use the VM's normal filesystem; sessions working in the same directory share its files and Git branch.

## 4. Attach to existing work

```bash
sess attach <name>
sess a <name>
```

Examples:

```bash
sess a api
sess a tests --host staging
```

Attaching reconnects to the existing terminal. It preserves the running shell's directory, environment, and programs.

If the session does not exist, sess reports that it is missing. **Attach never silently creates a fresh session.** Use `sess new <name>` when you want a new shell.

You can attach from another laptop with sess installed and SSH access to the same remote account. Set or explicitly supply the same destination; the session is already on the VM.

Multiple attached terminals share one shell and its input. They are not separate workspaces. Create separate named sessions when you want independent terminals.

## 5. Detach without stopping work

### Detach just this terminal

Press **Ctrl+\**: hold Control and press the backslash key.

The session continues running on the VM. Your local terminal returns to the shell from which you launched sess. If you attached through the sess browser, you return to the browser.

Intentional detach ends that attachment. sess does not immediately reconnect you.

### Close the terminal

You can also close the terminal tab or window. To return, open another terminal and run:

```bash
sess a api
```

### Detach from a shell command

At a shell prompt inside the session:

```bash
sess detach
```

This command detaches **all terminals attached to the current session**, matching the underlying `zmx detach` operation. It leaves the session running. Use **Ctrl+\** when you only want to detach your own terminal.

`sess detach` takes no session name or host flag and must be run inside a sess session. It does not terminate any running program. The equivalent advanced backend command is `zmx detach`. [zmx attachment and detach behaviour](https://github.com/neurosnap/zmx#usage).

### Detach, interrupt, and exit are different

| Action | Result |
| --- | --- |
| `Ctrl+\` | Detach this terminal; leave the session running |
| Close your local terminal | Disconnect; leave the remote session running |
| `sess detach` inside the session | Detach every attached terminal; leave the session running |
| `Ctrl+C` while attached | Send an interrupt to the foreground program, following normal terminal behaviour |
| `exit` at the session's main shell prompt | End that shell and session |
| `sess remove <name>` from your laptop | Terminate and remove the named session |

`Ctrl+D` at an empty shell prompt commonly exits the shell; behaviour depends on the shell and its settings. Exiting a nested shell returns to its parent instead of necessarily ending the session.

If an application needs `Ctrl+\`, zmx supports disabling its detach shortcut with `ZMX_NO_DETACH_KEY`. In that configuration, close the client terminal or use the detach command when a shell prompt is available. [zmx shortcut configuration](https://github.com/neurosnap/zmx#usage).

## 6. List sessions

```bash
sess ls
sess ls --host staging
```

Use `sess ls --json` for structured output or `sess ls --quiet` for session names only.

The list shows live sess-managed sessions on the selected host, including sessions with no attached terminals. For example:

```text
Host: dev

NAME          STATE      CLIENTS   AGE   DIRECTORY
api           attached   1         2h    /home/deepak
experiments   detached   0         5m    /home/deepak
tests         detached   0         1h    /home/deepak
```

These are illustrative values. The directory comes from zmx and may update when your shell reports directory changes; it is not guaranteed to track every `cd`. Zero clients means the session is running with nobody attached.

A session whose main shell has exited is no longer a live session. sess does not retain a stopped-session definition that can restore its processes.

If the host cannot be reached, sess reports that it cannot retrieve the list. It must not present an unreachable host as having no sessions.

## 7. Remove a session

```bash
sess remove <name>
sess rm <name>
```

Examples:

```bash
sess rm tests
sess rm experiments -h staging
```

Removal terminates the named session and the processes managed within it, disconnects its attached terminals, and removes its sess metadata. Unsaved work in those programs can be lost. Independently daemonized services are outside the session's lifecycle.

Files already written to the VM remain. Removal does not delete your project directory or undo changes to its files.

Use detach when you intend to return. Use remove when you are finished with that terminal.

Creating the same name afterward starts a fresh shell; it does not restore the removed processes.

## 8. Network loss and automatic reconnection

While an attachment is open, sess manages reconnection for you:

1. The SSH connection drops or becomes unresponsive.
2. sess displays the host, session name, and reconnecting status.
3. sess retries transient connection failures after 1, 2, 4, 8, then at most 15 seconds between attempts.
4. When SSH works again, sess attaches to the same existing session.

Press **Ctrl+C while the reconnect message is displayed** to stop retrying and return to your local shell. This does not remove the remote session.

A deliberate detach or normal shell exit ends the connection. Authentication failures, host-key problems, incompatible installations, and missing sessions produce an explanation rather than an endless retry loop.

If you close the terminal, quit sess, or reboot your laptop, its reconnect process is gone. Open a terminal and run `sess attach <name>` to return. No background laptop service is required.

An unreachable VM has an unknown session state. sess can confirm whether the session survived only after reaching the VM again.

## 9. What runs under the hood

```text
YOUR LAPTOP                         YOUR VM

Terminal application                sess remote helper
        |                                   |
   sess client  ===== SSH =====>       zmx client
                                            |
                                      Unix socket
                                            |
                                      zmx daemon
                                            |
                                     Virtual terminal
                                            |
                                      Shell + programs
```

### On your laptop

sess stores your default destination and runs session commands over SSH. During attachment, it manages connection status and reconnect attempts. The terminal application handles display, keyboard input, tabs, and windows.

### On the VM

The remote sess helper translates operations into zmx actions. zmx uses a daemon and Unix socket for each session and holds the shell through a virtual terminal, or PTY. It retains terminal state and scrollback with `libghostty-vt` for restoration on reattachment. [zmx architecture](https://github.com/neurosnap/zmx#impl).

The SSH connection can end while that daemon and shell continue. sess needs no additional network listener beyond the VM's SSH service.

### The command lifecycle

**New:** resolve the host → connect through SSH → check for a duplicate → create the persistent terminal → attach.

**Attach:** resolve the host → verify the named session exists → attach → supervise the connection until detach, exit, or failure.

**List:** resolve the host → request the remote session list → print it locally.

**Remove:** resolve the host → terminate the named session → remove its metadata → report the result.

The host owns the running session. Changing laptop settings cannot move it. sess enforces separate create and attach operations even though the backend's own attach command can create sessions.

## 10. What persistence means

| Event | What to expect |
| --- | --- |
| Detach or close the client terminal | Work continues on the VM |
| Laptop sleeps or loses its network | Work can continue on the VM; reconnect when the link returns |
| Laptop reboots | Remote work can continue; attach manually afterward |
| A program crashes inside the shell | That program ends; the shell may remain usable |
| The session's main shell exits | The session ends |
| VM reboots, loses power, or is destroyed | Live sessions and their processes are lost |
| zmx crashes or the OS kills its processes | Session persistence can be lost |

Persistence depends on the VM staying alive and allowing user processes to survive logout. It is not process checkpointing, a backup, or automatic job recovery after a VM reboot.

Restored terminal output is convenient working history. Save important output to files when you need a durable record.

Backend upgrades can also affect live sessions; incompatible zmx communication changes are a documented risk. sess setup preserves an existing supported backend and refuses an automatic change to a different managed version. [zmx known issues](https://github.com/neurosnap/zmx#known-issues).

## 11. Session browser and help

Run:

```bash
sess
```

In an interactive terminal, this opens the session browser for your default host. Use `sess --host staging` to browse another host. The browser lets you select and attach, create a session, or remove one. It uses the same rules as the commands above.

After attachment, the session uses the terminal directly. sess does not add windows, panes, splits, or a permanent status bar. For independent visible terminals, use separate tabs in your terminal application.

When output is redirected or no interactive terminal is available, bare `sess` prints the selected host's session list.

Additional commands:

```bash
sess doctor                 # Check laptop configuration and the selected VM
sess doctor -h staging      # Check a specific VM
sess --help                 # Show commands
sess new --help             # Show command-specific help
sess version                # Show the installed version
```

Start sess attachments from an ordinary laptop terminal. If already inside a persistent terminal, detach before starting another attachment; nested zmx sessions across SSH have documented display limitations. [zmx known issues](https://github.com/neurosnap/zmx#known-issues).

### Browser controls

| Key | Action |
| --- | --- |
| `↑` / `↓` or `k` / `j` | Move selection |
| `Enter` | Attach, or submit the current form |
| `n` | Create and attach |
| `x` | Remove the selected session after typing its name |
| `/` | Filter sessions by name |
| `h` | Browse another host; keep the saved default unchanged |
| `r` | Refresh the list |
| `?` | Show keyboard help |
| `Esc` | Cancel a form, or leave the browser |
| `q` | Leave the browser when not editing a form |

The browser requires a terminal at least 44 columns wide and 16 rows tall. It refreshes in the background. It hands the terminal directly to SSH while attached and returns on detach. Set `NO_COLOR` to disable its colors.

## 12. Configuration and shell completion

The default host is stored on your laptop in `~/.config/sess/config.json`, or under `XDG_CONFIG_HOME` when set. `SESS_CONFIG` overrides the complete configuration file path. It is JSON, written atomically with private permissions.

Normal users only need `sess set --host`. For development, `SESS_SSH` selects an SSH executable or wrapper, and the remote overrides `SESS_ZMX` and `SESS_RUNTIME_DIR` select a backend executable and private runtime directory. These are not forwarded automatically from your laptop to the VM.

Generate completion for your shell:

```sh
sess completion bash
sess completion zsh
sess completion fish
```

The commands print completion scripts. Install them using your shell's completion conventions; `make install` from source installs bash and zsh scripts under its prefix. Session-name completion queries the selected host with a short timeout.

The remote helper installed by `init` is a small internal program, not the full laptop CLI. Run management commands on your laptop. Inside a remote session, it provides `sess detach`. If your shell configuration overwrites PATH, the equivalent full-path command is:

```sh
~/.local/share/sess/bin/sess detach
```

## Command reference

| Command | Alias | Purpose |
| --- | --- | --- |
| `sess init <host>` | — | Prepare sess and zmx on the VM |
| `sess set --host <host>` | `sess set -h <host>` | Save the default host on this laptop |
| `sess new <name>` | `sess n <name>` | Create and attach |
| `sess attach <name>` | `sess a <name>` | Attach to an existing session |
| `sess remove <name>` | `sess rm <name>` | Terminate and remove a session |
| `sess ls` | — | List live sessions on the selected host |
| `sess detach` | — | From inside a session, detach all its clients |
| `sess` | — | Open the session browser |
| `sess doctor` | — | Check the selected host and dependencies |
| `sess --help` | — | Show help |
| `sess version` | — | Show version |
| `sess completion <shell>` | — | Generate a completion script |

`new`, `attach`, `remove`, `ls`, the browser, and `doctor` accept `--host <host>` or `-h <host>` without changing the saved default.

**Daily loop: create → work → detach → attach. Remove when finished.**

[Installation](../README.md#install) · [Troubleshooting](troubleshooting.md) · [Changelog](../CHANGELOG.md)
