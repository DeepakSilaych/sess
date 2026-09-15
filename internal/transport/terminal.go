package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// runTerminal keeps OpenSSH on a real PTY while inspecting local bracketed
// pastes. Output is copied unchanged; SSH still owns the remote terminal.
func (c *Client) runTerminal(ctx context.Context, cmd *exec.Cmd, notice io.Writer) error {
	size, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return err
	}
	master, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return err
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		master.Close()
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	local, cancel := context.WithCancel(ctx)
	defer cancel()
	resized := make(chan os.Signal, 1)
	signal.Notify(resized, syscall.SIGWINCH)
	defer signal.Stop(resized)
	resizeDone := make(chan struct{})
	go func() {
		defer close(resizeDone)
		for {
			select {
			case <-local.Done():
				return
			case <-resized:
				pty.InheritSize(os.Stdin, master)
			}
		}
	}()
	outputDone := make(chan struct{})
	go func() { defer close(outputDone); io.Copy(os.Stdout, master) }()

	var uploadMu sync.Mutex
	var uploadCancel context.CancelFunc
	chunks := make(chan []byte, 64)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer close(chunks)
		buf := make([]byte, 4096)
		for local.Err() == nil {
			fds := []unix.PollFd{{Fd: int32(os.Stdin.Fd()), Events: unix.POLLIN}}
			n, e := unix.Poll(fds, 100)
			if e == unix.EINTR {
				continue
			}
			if e != nil {
				return
			}
			if n == 0 {
				continue
			}
			if local.Err() != nil {
				return
			}
			n, e = unix.Read(int(os.Stdin.Fd()), buf)
			if e != nil || n == 0 {
				return
			}
			b := append([]byte(nil), buf[:n]...)
			uploadMu.Lock()
			if uploadCancel != nil && bytes.Contains(b, []byte{3}) {
				uploadCancel()
				b = bytes.ReplaceAll(b, []byte{3}, nil)
			}
			uploadMu.Unlock()
			if len(b) == 0 {
				continue
			}
			select {
			case chunks <- b:
			case <-local.Done():
				return
			}
		}
	}()
	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		filter := pasteFilter{
			transform: func(b []byte) ([]byte, error) {
				transferCtx, stop := context.WithCancel(local)
				uploadMu.Lock()
				uploadCancel = stop
				uploadMu.Unlock()
				result, e := c.imagePaste(transferCtx, b)
				uploadMu.Lock()
				uploadCancel = nil
				uploadMu.Unlock()
				stop()
				return result, e
			},
			write:  func(b []byte) error { _, e := master.Write(b); return e },
			notice: func(e error) { fmt.Fprintf(notice, "\r\nsess: %s\r\n", e) },
		}
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-local.Done():
				return
			case b, ok := <-chunks:
				if !ok {
					return
				}
				if filter.feed(b) != nil {
					return
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(100 * time.Millisecond)
			case <-timer.C:
				if filter.idle() != nil {
					return
				}
				timer.Reset(100 * time.Millisecond)
			}
		}
	}()
	err = cmd.Wait()
	cancel()
	<-readerDone
	<-inputDone
	<-resizeDone
	// The slave closes when SSH exits; drain the remaining display before restoring.
	<-outputDone
	master.Close()
	return err
}
