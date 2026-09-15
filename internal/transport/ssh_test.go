package transport

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRetryClassification(t *testing.T) {
	for _, s := range []string{"Connection reset by peer", "Connection to 127.0.0.1 closed by remote host.", "ssh: connect to host dev port 22: Connection refused", "client_loop: send disconnect: Broken pipe"} {
		if !Transient(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"Permission denied (publickey).", "Host key verification failed.", "Could not resolve hostname dev", "Bad configuration option", "exit status 255"} {
		if Transient(s) {
			t.Fatal(s)
		}
	}
	if Backoff(0) != time.Second || Backoff(50) != 15*time.Second {
		t.Fatal("unbounded retry")
	}
}
func TestSSHArguments(t *testing.T) {
	c, _ := New("user@dev")
	cmd := c.Command(context.Background(), true, true, "exec helper")
	joined := strings.Join(cmd.Args, " ")
	for _, part := range []string{"-t", "BatchMode=yes", "ServerAliveInterval=10", "user@dev exec helper"} {
		if !strings.Contains(joined, part) {
			t.Fatal(joined)
		}
	}
}
