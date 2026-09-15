# Troubleshooting

Start with:

```sh
sess doctor
# Or inspect a specific destination:
sess doctor --host dev
```

## No default host configured

Choose a host after preparing it:

```sh
sess init dev
sess set --host dev
```

Or use `--host dev` for one command. `init` does not set the default. Host overrides do not change it.

## SSH works interactively, but sess fails

Session operations run SSH in batch mode so list refreshes and reconnect attempts cannot ask for passwords. Test the same mode:

```sh
ssh -o BatchMode=yes dev true
```

Configure SSH key authentication and load an encrypted key into your SSH agent as needed:

```sh
ssh-add ~/.ssh/id_ed25519
```

Use your actual key path. Check that the alias points to the expected account, hostname, port, and jump host. See `ssh dev` for interactive diagnostics. Host-key verification failures require resolving the host's identity through your normal SSH process; sess does not bypass them.

## Invalid SSH destination

Use `user@host`, a hostname, or an SSH alias. `user:host`, command options, and SSH URLs are not accepted. Put ports and identities in `~/.ssh/config`.

`-h` selects the host. Use `--help` for help.

## sess is not installed on the host

Run `sess init <host>` from your laptop. It installs a small remote helper and the pinned backend under `~/.local/share/sess/bin`. A successful local installation does not prepare the VM automatically.

## Unexpected response from the host

Run `sess init <host>` to check the remote helper. Also check shell startup files: non-interactive SSH commands must not print banners or other text to stdout before the helper's JSON response. Human greetings belong in interactive-only shell startup paths.

## Session not found

List sessions on the intended host and account:

```sh
sess ls -h dev
```

A session disappears when its main shell exits, it is removed, or its backend is lost. `sess attach` does not recreate it. Use `sess new <name>` for a new shell. A different SSH account on the same machine has different sessions.

## A session exists, but nobody is attached

That is normal. `STATE=detached` or `CLIENTS=0` means the terminal is still running with no attached clients. Attach to return to it.

## Attachment requires an interactive terminal

Run `attach` from a terminal. For automation, create without attaching:

```sh
sess new build --detach
sess ls --json
```

Redirected bare `sess` prints a list instead of opening the browser.

## Reconnect stopped after a session was replaced

The original session was removed and its name reused. sess checks the session identity and stops reconnecting to avoid taking you into different work. Run `sess attach <name>` explicitly if you want the new session.

## The terminal is reconnecting

If the laptop still has the sess process open, transient SSH failures retry with a delay capped at 15 seconds. Press Ctrl+C during the reconnect message to stop trying. This leaves the remote session alone.

If the terminal was closed or the laptop rebooted, open another terminal and run `sess attach <name>`. sess does not run a background reconnect service on your laptop.

## Ctrl+\ conflicts with an application

zmx handles Ctrl+\ as its detach shortcut by default. Its `ZMX_NO_DETACH_KEY` environment option can disable that behaviour, but a variable set only on your laptop is not automatically forwarded to the remote attachment. Backend configuration must reach the remote process.

You can close your local terminal to disconnect. At a shell prompt, `sess detach` disconnects all clients of the current session. `exit` ends the shell instead of detaching.

## sess detach is not found inside the session

The session starts with the managed binary directory on PATH. Your shell configuration may replace PATH. Use:

```sh
~/.local/share/sess/bin/sess detach
```

The installed remote program is a helper, not the full client. Run management commands such as `new`, `ls`, and `set` on your laptop.

## Unsupported zmx version

sess 0.7.0 is paired with zmx 0.8.1. It deliberately refuses to operate with a different managed backend version or silently replace it. Backend changes can affect live sessions. Finish that work before changing the backend installation.

An ordinary zmx installation uses a separate namespace and is not managed by sess. Do not delete runtime socket files to terminate sessions; use the lifecycle commands.

## Wrong or stale directory in the list

The displayed directory is supplied by zmx. It starts at the creation directory and may update when the shell emits directory information through OSC 7. Shells that do not report changes can leave the display at its earlier value. Run `pwd` inside the session for its current working directory.

## I upgraded from tmux-based sess

Your old tmux sessions remain in tmux. For example:

```sh
tmux ls
tmux attach -t work
```

The new client does not import `~/.sess` or convert running processes. Prepare the host with `sess init`, set your default host, and create new zmx sessions when ready. See the [0.6.0 migration notes](../CHANGELOG.md#migration-from-05).

## Report a problem

Include `sess version`, the command, the selected platform, whether the problem involves a TUI attachment or plain command, and the relevant `sess doctor` output. Remove hostnames or account information you do not want public. Never include private keys or credentials.

[Open an issue](https://github.com/DeepakSilaych/sess/issues) · [User guide](user-guide.md)

## Image drops and uploads

See [file transfer troubleshooting](file-transfer.md#troubleshooting) for terminal paste compatibility, remote helper updates, and upload errors.
