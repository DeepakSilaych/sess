# Contributing to sess

sess manages named persistent SSH terminals. The remote machine runs the programs; zmx owns their terminal; the client provides commands, a browser, and reconnection.

## Set up a checkout

Install Go 1.26.5 or newer, Make, and OpenSSH. Docker and Python 3 are needed for the SSH integration suite.

```sh
git clone https://github.com/DeepakSilaych/sess.git
cd sess
make build
./dist/sess --help
```

`make build` generates compressed remote helpers for macOS/Linux on AMD64/ARM64 before building the client. The generated helpers and `dist/` are ignored by Git. The `bin/sess` script is a convenience launcher for a source checkout, not the release executable.

## Project structure

| Directory | Responsibility |
| --- | --- |
| `cmd/sess` | Client entry point |
| `cmd/sess-agent` | Small remote-helper entry point |
| `internal/api` | Shared request/response types, version constants, and validation |
| `internal/cli` | Commands, help, output formats, and completion |
| `internal/tui` | Browser state, rendering, and keyboard interaction |
| `internal/store` | Local host configuration |
| `internal/transport` | System SSH and reconnect policy |
| `internal/backend` | zmx commands, runtime isolation, and session lifecycle |
| `internal/agent` | Remote request handling |
| `internal/provision` | Embedded helpers and verified backend installation |
| `test/integration` | Disposable Docker SSH fixtures and PTY tests |

The browser and commands use the same remote operations. New UI features must preserve that behaviour. Use the system SSH client so users' SSH aliases, keys, and jump-host settings continue to work.

## Verify a change locally

```sh
make test                # go test -race ./... and go vet ./...
make integration         # build, then test real SSH and zmx in Docker
```

The integration suite uses temporary keys and configuration, publishes its SSH port only on loopback, and removes its container on exit. It does not use your configured sess hosts. The suite needs internet access for its Docker image and the pinned zmx release.

Optional native-backend validation:

```sh
SESS_TEST_ZMX=/absolute/path/to/verified/zmx go test ./internal/backend -run TestNativeZMX -v
```

Use zmx 0.8.1. The native test uses a temporary home and private runtime directory and removes its test session afterward.

There are no GitHub Actions workflows. Run checks locally and include results in your change description. Do not add CI as part of unrelated changes.

## Change conventions

- Keep host selection consistent: explicit flag, then saved default, then an error.
- Keep `-h` for host and `--help` for help.
- Validate session names and SSH destinations before acting on them.
- Preserve the distinction between create and attach, and between detach and session termination.
- Avoid silently retrying authentication, host-key, or configuration errors.
- Do not interpret shell output as executable configuration or interpolate untrusted arguments into shell commands.
- Treat an unreachable host as unknown, not empty.
- Keep terminal handoff and restoration correct on success, error, resize, and cancellation.
- Keep new files formatted with `gofmt`, and add tests for changed lifecycle behaviour.

Update the README, user guide, landing page, and changelog when commands or behaviour change. Generate completion files after changes to the command tree:

```sh
make build
./dist/sess completion bash > etc/bash-completion/sess
./dist/sess completion zsh > etc/zsh-completion/_sess
```

See [the manual release process](docs/releasing.md) for version tags and downloadable builds.
