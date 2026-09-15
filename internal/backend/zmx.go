// Package backend isolates sess from zmx's command and runtime details.
package backend

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ZMX struct{ Binary, Runtime, Home string }

func Open() (*ZMX, error) {
	h, e := os.UserHomeDir()
	if e != nil {
		return nil, e
	}
	bin := os.Getenv("SESS_ZMX")
	if bin == "" {
		bin = filepath.Join(h, ".local/share/sess/bin/zmx")
	}
	runtime := os.Getenv("SESS_RUNTIME_DIR")
	if runtime == "" {
		sum := sha256.Sum256([]byte(h))
		runtime = fmt.Sprintf("/tmp/sess-%d-%x", os.Getuid(), sum[:4])
	}
	if e = os.MkdirAll(runtime, 0700); e != nil {
		return nil, e
	}
	st, e := os.Lstat(runtime)
	if e != nil {
		return nil, e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("sess runtime must be a private directory: %s", runtime)
	}
	if stat, ok := st.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		return nil, fmt.Errorf("sess runtime has a different owner")
	}
	return &ZMX{bin, runtime, h}, nil
}
func (z *ZMX) Env(name string) []string {
	env := []string{}
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "ZMX_") || strings.HasPrefix(v, "SESS_SESSION=") || strings.HasPrefix(v, "SESS_RUNTIME_DIR=") || strings.HasPrefix(v, "PATH=") {
			continue
		}
		env = append(env, v)
	}
	env = append(env, "PATH="+filepath.Join(z.Home, ".local/share/sess/bin")+":"+os.Getenv("PATH"), "ZMX_DIR="+z.Runtime, "ZMX_DIR_MODE=0700", "ZMX_LOG_MODE=0600", "SESS_RUNTIME_DIR="+z.Runtime)
	if os.Getenv("ZMX_NO_DETACH_KEY") != "" {
		env = append(env, "ZMX_NO_DETACH_KEY=1")
	}
	if name != "" {
		env = append(env, "SESS_SESSION="+name)
	}
	return env
}
func (z *ZMX) command(ctx context.Context, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, z.Binary, args...)
	c.Env = z.Env("")
	c.Dir = z.Home
	return c
}
func (z *ZMX) Version(ctx context.Context) (string, error) {
	b, e := z.command(ctx, "version").CombinedOutput()
	if e != nil {
		return "", api.Fail("backend", "zmx is not available.", "Run sess init <host> to prepare this VM.")
	}
	line := strings.Split(strings.TrimSpace(string(b)), "\n")[0]
	// Pin the CLI output contract; never operate on an untested IPC version.
	fields := strings.Fields(line)
	found := false
	for _, f := range fields {
		if strings.TrimPrefix(f, "v") == api.ZMXVersion {
			found = true
		}
	}
	if !found {
		return line, api.Fail("backend_version", "Unsupported zmx version: "+line, "sess requires zmx "+api.ZMXVersion+". Keep existing sessions running and update after finishing them.")
	}
	return line, nil
}
func ParseList(raw string) ([]api.Session, error) {
	sessions := []api.Session{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		values := map[string]string{}
		for _, part := range strings.Split(line, "\t") {
			k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
			if ok {
				values[k] = v
			}
		}
		name := values["name"]
		if e := api.ValidateName(name); e != nil {
			return nil, fmt.Errorf("unexpected zmx list output")
		}
		if values["err"] != "" {
			return nil, fmt.Errorf("zmx session %s is unreachable: %s", name, values["err"])
		}
		clients, e := strconv.Atoi(values["clients"])
		if e != nil {
			return nil, fmt.Errorf("invalid zmx client count")
		}
		pid, e := strconv.Atoi(values["pid"])
		if e != nil {
			return nil, e
		}
		created, e := strconv.ParseInt(values["created"], 10, 64)
		if e != nil {
			return nil, e
		}
		status := "detached"
		if clients > 0 {
			status = "attached"
		}
		directory := values["cwd"]
		if strings.HasPrefix(directory, "file://") {
			if u, e := url.Parse(directory); e == nil {
				directory = u.Path
			}
		}
		sessions = append(sessions, api.Session{Name: name, ID: values["sess_id"], Clients: clients, PID: pid, Created: time.Unix(created, 0).UTC(), Directory: directory, Status: status})
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Name < sessions[j].Name })
	return sessions, nil
}
func (z *ZMX) List(ctx context.Context) ([]api.Session, error) {
	c := z.command(ctx, "list")
	var stderr strings.Builder
	c.Stderr = &stderr
	b, e := c.Output()
	if e != nil {
		return nil, fmt.Errorf("list sessions: %s: %w", strings.TrimSpace(stderr.String()), e)
	}
	return ParseList(string(b))
}
func (z *ZMX) Find(ctx context.Context, name, id string) (api.Session, error) {
	if e := api.ValidateName(name); e != nil {
		return api.Session{}, e
	}
	ss, e := z.List(ctx)
	if e != nil {
		return api.Session{}, e
	}
	for _, s := range ss {
		if s.Name == name {
			if id != "" && id != s.ID {
				return s, api.Fail("replaced", "Session was replaced: "+name, "Run sess attach "+name+" to connect to the new session explicitly.")
			}
			return s, nil
		}
	}
	return api.Session{}, api.Fail("not_found", "Session not found: "+name, "Run sess ls to see sessions, or sess new "+name+" to create one.")
}
func (z *ZMX) lock(name string) (func(), error) {
	if e := api.ValidateName(name); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(filepath.Join(z.Runtime, "."+name+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, api.Fail("busy", "Session operation already in progress: "+name, "Try again in a moment.")
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
func (z *ZMX) Create(ctx context.Context, name string) (api.Session, error) {
	unlock, e := z.lock(name)
	if e != nil {
		return api.Session{}, e
	}
	defer unlock()
	ss, e := z.List(ctx)
	if e != nil {
		return api.Session{}, e
	}
	for _, s := range ss {
		if s.Name == name {
			return s, api.Fail("exists", "Session already exists: "+name, "Use sess attach "+name+".")
		}
	}
	var id [16]byte
	if _, e = rand.Read(id[:]); e != nil {
		return api.Session{}, e
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	c := z.command(ctx, "attach", "--labels", "sess_id="+hex.EncodeToString(id[:]), name, shell, "-i")
	c.Env = z.Env(name)
	// zmx supports a non-TTY client. An explicit detach byte leaves the newly
	// created PTY alive, so a dropped SSH connection cannot create it twice.
	c.Stdin = strings.NewReader("\x1c")
	c.Stdout = io.Discard
	var stderr strings.Builder
	c.Stderr = &stderr
	if e = c.Run(); e != nil {
		return api.Session{}, fmt.Errorf("create session: %s: %w", stderr.String(), e)
	}
	return z.Find(ctx, name, "")
}
func (z *ZMX) Remove(ctx context.Context, name, id string) error {
	unlock, e := z.lock(name)
	if e != nil {
		return e
	}
	defer unlock()
	if _, e = z.Find(ctx, name, id); e != nil {
		return e
	}
	b, e := z.command(ctx, "kill", name).CombinedOutput()
	if e != nil {
		return fmt.Errorf("remove session: %s: %w", string(b), e)
	}
	return nil
}
func (z *ZMX) Attach(ctx context.Context, name, id string) error {
	if _, e := z.Find(ctx, name, id); e != nil {
		return e
	}
	// zmx attach is an upsert. If the session exits after the check, /usr/bin/false
	// prevents the backend from accidentally opening a replacement shell.
	c := z.command(ctx, "attach", name, "/usr/bin/false")
	c.Env = z.Env(name)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
func (z *ZMX) Detach(ctx context.Context) error {
	name := os.Getenv("SESS_SESSION")
	if e := api.ValidateName(name); e != nil {
		return api.Fail("outside_session", "Not inside a sess session.", "Press Ctrl+\\ in an attached terminal, or run sess detach inside its shell.")
	}
	c := z.command(ctx, "detach")
	c.Env = append(z.Env(name), "ZMX_SESSION="+name)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
