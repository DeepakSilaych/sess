// Package transfer implements bounded, atomic uploads to the remote account.
package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const MaxSize = 25 * 1024 * 1024

// Receive never publishes an incomplete file and never overwrites a prior upload.
func Receive(home, name string, size int64, digest string, in io.Reader) (string, error) {
	if size < 0 || size > MaxSize {
		return "", fmt.Errorf("uploads must be at most 25 MiB")
	}
	if name == "" || name == "." || name == ".." || len(name) > 200 || strings.ContainsAny(name, "/\\\x00\r\n\x1b") {
		return "", fmt.Errorf("invalid upload filename")
	}
	for _, r := range name {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("invalid upload filename")
		}
	}
	hash, err := hex.DecodeString(digest)
	if err != nil || len(hash) != sha256.Size {
		return "", fmt.Errorf("invalid upload checksum")
	}
	root := filepath.Join(home, ".local", "share", "sess", "uploads")
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(root, "upload-")
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		if !complete {
			os.RemoveAll(dir)
		}
	}()
	// Keep the original name for Claude and use a random parent for collision isolation.
	dst := filepath.Join(dir, name)
	f, err := os.OpenFile(filepath.Join(dir, ".partial"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(in, size+1))
	if err != nil {
		return "", err
	}
	if n != size {
		return "", fmt.Errorf("upload size mismatch")
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return "", fmt.Errorf("upload checksum mismatch")
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(filepath.Join(dir, ".partial"), dst); err != nil {
		return "", err
	}
	complete = true
	return dst, nil
}
