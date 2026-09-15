package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"golang.org/x/term"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const AgentPath = `"$HOME/.local/share/sess/bin/sess"`

type Client struct{ Host string }

func New(host string) (*Client, error) {
	if e := api.ValidateHost(host); e != nil {
		return nil, e
	}
	return &Client{host}, nil
}
func (c *Client) Command(ctx context.Context, tty, batch bool, remote string) *exec.Cmd {
	bin := os.Getenv("SESS_SSH")
	if bin == "" {
		bin = "ssh"
	}
	args := []string{"-o", "ConnectTimeout=8", "-o", "ServerAliveInterval=10", "-o", "ServerAliveCountMax=3"}
	if tty {
		args = append(args, "-t")
	} else {
		args = append(args, "-T")
	}
	if batch {
		args = append(args, "-o", "BatchMode=yes")
	}
	args = append(args, c.Host, remote)
	return exec.CommandContext(ctx, bin, args...)
}
func (c *Client) Request(ctx context.Context, r api.Request) (api.Response, error) {
	var response api.Response
	cmd := c.Command(ctx, false, true, "exec "+AgentPath+" rpc "+api.Encode(r))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, e := cmd.Output()
	if e != nil {
		return response, ConnectionError(c.Host, stderr.String(), e)
	}
	if e = json.Unmarshal(b, &response); e != nil {
		return response, api.Fail("protocol", "Unexpected response from "+c.Host+".", "Run sess init "+c.Host+". Ensure non-interactive shell startup files do not print to stdout.")
	}
	if response.Protocol != api.Protocol {
		return response, api.Fail("protocol", "Remote sess protocol is incompatible.", "Run sess init "+c.Host+".")
	}
	if response.Error != nil {
		return response, response.Error
	}
	return response, nil
}
func (c *Client) List(ctx context.Context) ([]api.Session, error) {
	r, e := c.Request(ctx, api.Request{Action: "list"})
	if r.Sessions == nil {
		r.Sessions = []api.Session{}
	}
	return r.Sessions, e
}
func (c *Client) Find(ctx context.Context, name string) (api.Session, error) {
	r, e := c.Request(ctx, api.Request{Action: "find", Name: name})
	if e != nil {
		return api.Session{}, e
	}
	if r.Session == nil {
		return api.Session{}, fmt.Errorf("remote did not return a session")
	}
	return *r.Session, nil
}
func (c *Client) Create(ctx context.Context, name string) (api.Session, error) {
	r, e := c.Request(ctx, api.Request{Action: "create", Name: name})
	if e != nil {
		return api.Session{}, e
	}
	if r.Session == nil {
		return api.Session{}, fmt.Errorf("remote did not return a session")
	}
	return *r.Session, nil
}
func (c *Client) Remove(ctx context.Context, s api.Session) error {
	_, e := c.Request(ctx, api.Request{Action: "remove", Name: s.Name, ID: s.ID})
	return e
}

// Do not retry generic 255 exits. OpenSSH uses 255 for authentication, host-key
// and configuration errors as well as transport failures.
func Transient(detail string) bool {
	s := strings.ToLower(detail)
	for _, permanent := range []string{"permission denied", "host key verification failed", "remote host identification has changed", "bad configuration", "bad owner", "could not resolve hostname", "no matching", "too many authentication failures", "no such file or directory"} {
		if strings.Contains(s, permanent) {
			return false
		}
	}
	for _, transient := range []string{"connection timed out", "operation timed out", "connection refused", "connection reset", "broken pipe", "connection closed", "closed by remote host", "network is unreachable", "no route to host", "timeout, server", "software caused connection abort"} {
		if strings.Contains(s, transient) {
			return true
		}
	}
	return false
}
func ConnectionError(host, detail string, cause error) error {
	detail = strings.TrimSpace(detail)
	if strings.Contains(detail, ".local/share/sess/bin/sess") && strings.Contains(detail, "not found") {
		return api.Fail("not_initialized", "sess is not installed on "+host+".", "Run sess init "+host+".")
	}
	if detail == "" {
		detail = cause.Error()
	}
	return api.Fail("ssh", "Cannot connect to "+host+": "+detail, "Check ssh "+host+". For non-interactive operations, unlock your SSH key in ssh-agent.")
}
func Backoff(attempt int) time.Duration {
	d := time.Second << min(attempt, 4)
	return min(d, 15*time.Second)
}
func (c *Client) Attach(ctx context.Context, s api.Session, notice io.Writer) error {
	if s.ID == "" {
		return api.Fail("identity", "Session has no sess identity.", "Create a fresh session using sess new.")
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, e := term.GetState(int(os.Stdin.Fd()))
		if e == nil {
			defer term.Restore(int(os.Stdin.Fd()), state)
		}
	}
	for attempt := 0; ; attempt++ {
		cmd := c.Command(ctx, true, true, "exec "+AgentPath+" attach "+api.Encode(api.Request{Action: "attach", Name: s.Name, ID: s.ID}))
		var detail bytes.Buffer
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(notice, &detail)
		e := cmd.Run()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e == nil {
			return nil
		} // explicit detach or remote shell exit
		var ex *exec.ExitError
		if !errors.As(e, &ex) || ex.ExitCode() != 255 || !Transient(detail.String()) {
			if errors.As(e, &ex) && ex.ExitCode() == 40 {
				return api.Fail("attach", "Remote attachment ended with an error.", "Run sess ls --host "+c.Host+" to check the session.")
			}
			return ConnectionError(c.Host, detail.String(), e)
		}
		delay := Backoff(attempt)
		fmt.Fprintf(notice, "\nReconnecting to %s / %s in %s · Ctrl+C to cancel\n", c.Host, s.Name, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
