# Images and file transfer

sess 0.7.0 can upload local files through the same system SSH configuration used by your sessions. Update the remote helper with `sess init <host>` after upgrading your laptop client. zmx stays at 0.8.1.

## Drop an image into Claude Code

1. Attach with `sess a work` and run Claude Code on the VM.
2. Drag a local PNG, JPEG, GIF, or WebP file into the terminal while the prompt is ready.
3. If the terminal sends a bracketed paste, sess replaces the local image path with an uploaded remote path.
4. Continue typing and press Enter when ready. sess does not submit your prompt.

The entire paste must contain only absolute local image paths. Spaces, single/double quotes, escaped spaces, and multiple filenames are supported. At most eight images may be uploaded in one paste, totaling 25 MiB. The file contents must match a supported image type. Ordinary text and unsupported pastes pass through unchanged.

An upload failure leaves out the image path and reports an error. Ctrl+C during a transfer cancels it; Ctrl+C during ordinary terminal use continues to reach the remote application. Earlier completed files from a multi-image upload remain on the VM if a later file fails.

## Terminal compatibility

A bracketed paste surrounds pasted text with standard terminal markers. The remote application enables this mode, and the local terminal decides whether file drops use it. sess processes those markers without an iTerm2, Ghostty, or Terminal.app API.

Some terminal/version combinations send drops as plain keystrokes, even when ordinary text pastes are bracketed. sess cannot reliably distinguish those paths from typing, so it leaves them unchanged. If a drop displays a local `/Users/...` path, use the explicit upload command below. Copying an image to the clipboard is different from pasting its file path; clipboard pixel transfer is not implemented.

Validation covers fragmented bracketed input over a real SSH connection, quoted filenames, multiple images, cancellation, resize, detach, and reconnect. This does not establish that every terminal application's native drag behavior or every Claude Code version has been tested.

## Upload explicitly

Run on your laptop, in another terminal if your current one is attached:

```sh
sess upload ~/Desktop/screenshot.png
sess upload ./first.png ./second.png --host dev
sess upload ./report.pdf
```

Each successful upload prints one absolute remote path to stdout. Copy that path into the remote application. Errors go to stderr and return a nonzero exit status. Files already uploaded are retained if a later file fails. The command accepts any regular file up to 25 MiB, uses the saved host or `--host`/`-h`, and does not require a session.

## Storage and privacy

Files are stored under the remote account's `~/.local/share/sess/uploads/upload-<unique>/` directory. A unique directory prevents duplicate names from overwriting earlier files. New upload directories use mode 0700 and files use 0600. The receiver checks both byte count and SHA-256 digest before renaming the temporary file and returning its path.

Images travel over a separate ordinary SSH connection to the same host. SSH aliases, keys, ports, and jump hosts still come from your SSH configuration. There is no upload server, public URL, or extra port.

Uploads are account files, independent of session names. They survive detaching, removing a session, and rebooting the VM. Delete individual upload directories on the VM when you no longer need them. No automatic cleanup is applied because Claude Code or another program may still reference an earlier image.

## Disable automatic uploads

```sh
sess a work --no-upload-images
sess new work --no-upload-images
sess --no-upload-images
```

This restores direct SSH attachment and passes all pasted paths through unchanged. Explicit `sess upload` remains available.

## Troubleshooting

- **The local path appears unchanged:** the terminal may not mark drops as bracketed pastes, or the paste contains prose, a relative path, an unsupported image, an unreadable file, or files over the limit. Use `sess upload` for a clear file error or remote path.
- **The remote helper rejects upload:** run `sess init <host>` from the updated client. Existing sessions keep their zmx backend.
- **Upload fails or times out:** verify SSH access and free space on the VM. Transfers time out after 60 seconds. No failed local path is inserted automatically.
- **Claude Code shows a path rather than an image badge:** the image exists at that remote path. Ask Claude Code to read it; inline rendering depends on its version and UI behavior.
