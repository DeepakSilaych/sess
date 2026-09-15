package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"github.com/DeepakSilaych/sess/internal/transport"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed assets/*
var assets embed.FS
var digests = map[string]string{
	"linux-arm64":  "943eb44c812333fd450da12097521afd3339436e86f8c2ac618b905c4c9ece68",
	"linux-amd64":  "dfd75720b942466f28870731cc86dbc07afa72fb8f3bd5eeb4ff707e4eecebe8",
	"darwin-arm64": "1d86b1c9fba47fa707a6f0e976b20510b07c1c26d0ed010b9414b2a2c5e6beef",
	"darwin-amd64": "3208578ad91d8a62077772dc8a1369a92033d9e84169bac673ef8542b6ff9707",
}

func Platform(raw string) (string, error) {
	f := strings.Fields(raw)
	if len(f) != 2 {
		return "", fmt.Errorf("cannot detect remote platform (shell startup must not print output)")
	}
	osname := map[string]string{"Linux": "linux", "Darwin": "darwin"}[f[0]]
	arch := map[string]string{"aarch64": "arm64", "arm64": "arm64", "x86_64": "amd64"}[f[1]]
	p := osname + "-" + arch
	if _, ok := digests[p]; !ok {
		return "", fmt.Errorf("unsupported VM platform: %s %s", f[0], f[1])
	}
	return p, nil
}
func Agent(platform string) ([]byte, error) {
	data, e := assets.ReadFile("assets/" + platform + ".gz")
	if e != nil {
		return nil, fmt.Errorf("this sess build has no %s remote helper; build with make build or use a release archive", platform)
	}
	r, e := gzip.NewReader(bytes.NewReader(data))
	if e != nil {
		return nil, e
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, 32<<20))
}
func Backend(ctx context.Context, platform string) ([]byte, error) {
	p := strings.Split(platform, "-")
	osname := p[0]
	if osname == "darwin" {
		osname = "macos"
	}
	arch := "x86_64"
	if p[1] == "arm64" {
		arch = "aarch64"
	}
	url := fmt.Sprintf("https://github.com/neurosnap/zmx/releases/download/v%s/zmx-%s-%s-%s.tar.gz", api.ZMXVersion, api.ZMXVersion, osname, arch)
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if e != nil {
		return nil, e
	}
	client := http.Client{Timeout: 90 * time.Second}
	res, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("download zmx: %s", res.Status)
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if e != nil {
		return nil, e
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != digests[platform] {
		return nil, fmt.Errorf("zmx archive checksum does not match pinned release")
	}
	return ExtractZMX(data)
}
func ExtractZMX(data []byte) ([]byte, error) {
	gz, e := gzip.NewReader(bytes.NewReader(data))
	if e != nil {
		return nil, e
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if path.Base(h.Name) == "zmx" && h.Typeflag == tar.TypeReg {
			if h.Size > 32<<20 {
				return nil, fmt.Errorf("zmx binary too large")
			}
			return io.ReadAll(io.LimitReader(tr, 32<<20))
		}
	}
	return nil, fmt.Errorf("zmx archive contains no executable")
}
func bundle(agent, zmx []byte) ([]byte, error) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for _, f := range []struct {
		name string
		data []byte
	}{{"sess", agent}, {"zmx", zmx}} {
		if e := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0755, Size: int64(len(f.data))}); e != nil {
			return nil, e
		}
		if _, e := tw.Write(f.data); e != nil {
			return nil, e
		}
	}
	if e := tw.Close(); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}

const installScript = `set -eu
umask 077
base="$HOME/.local/share/sess"
mkdir -p "$base/bin" "$HOME/.local/bin"
staging=$(mktemp -d "$base/.install.XXXXXXXX")
trap 'rm -rf "$staging"' EXIT HUP INT TERM
tar -xf - -C "$staging"
if [ -x "$base/bin/zmx" ]; then
  if ! "$base/bin/zmx" version | head -n 1 | grep -Eq '(^|[[:space:]])v?0[.]8[.]1([[:space:]]|$)'; then
    echo 'Existing sess zmx version differs. Finish active sessions before changing the backend.' >&2
    exit 1
  fi
else
  mv "$staging/zmx" "$base/bin/zmx"
fi
mv "$staging/sess" "$base/bin/sess"
# Never overwrite a separately installed sess command.
if [ ! -e "$HOME/.local/bin/sess" ] && [ ! -L "$HOME/.local/bin/sess" ]; then
  ln -s "$base/bin/sess" "$HOME/.local/bin/sess"
fi
`

func Init(ctx context.Context, c *transport.Client, out io.Writer) error {
	fmt.Fprintf(out, "[1/4] Connecting to %s\n", c.Host)
	cmd := c.Command(ctx, false, false, "uname -s; uname -m")
	cmd.Stdin = os.Stdin
	cmd.Stderr = out
	raw, e := cmd.Output()
	if e != nil {
		return fmt.Errorf("SSH check failed: %w", e)
	}
	p, e := Platform(string(raw))
	if e != nil {
		return e
	}
	fmt.Fprintf(out, "[2/4] Preparing %s helper and zmx %s\n", p, api.ZMXVersion)
	agent, e := Agent(p)
	if e != nil {
		return e
	}
	zmx, e := Backend(ctx, p)
	if e != nil {
		return e
	}
	archive, e := bundle(agent, zmx)
	if e != nil {
		return e
	}
	fmt.Fprintln(out, "[3/4] Installing for your remote account")
	cmd = c.Command(ctx, false, true, "sh -c "+shellQuote(installScript))
	cmd.Stdin = bytes.NewReader(archive)
	cmd.Stdout = out
	cmd.Stderr = out
	if e = cmd.Run(); e != nil {
		return fmt.Errorf("remote installation failed: %w", e)
	}
	fmt.Fprintln(out, "[4/4] Verifying remote session support")
	_, e = c.Request(ctx, api.Request{Action: "ping"})
	if e != nil {
		return e
	}
	fmt.Fprintf(out, "\nReady: %s\n\nSet your default host:\n  sess set --host %s\n", c.Host, c.Host)
	return nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
