package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/DeepakSilaych/sess/internal/api"
	"github.com/DeepakSilaych/sess/internal/transfer"
)

func readUpload(name string, imageOnly bool) ([]byte, error) {
	f, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("upload requires a regular file")
	}
	if st.Size() > transfer.MaxSize {
		return nil, fmt.Errorf("upload exceeds 25 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, transfer.MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > transfer.MaxSize {
		return nil, fmt.Errorf("upload exceeds 25 MiB")
	}
	if imageOnly {
		switch http.DetectContentType(data) {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
		default:
			return nil, fmt.Errorf("automatic uploads support PNG, JPEG, GIF and WebP images")
		}
	}
	return data, nil
}

func (c *Client) Upload(ctx context.Context, name string) (string, error) {
	data, err := readUpload(name, false)
	if err != nil {
		return "", err
	}
	return c.uploadBytes(ctx, filepath.Base(name), data)
}

func (c *Client) uploadBytes(ctx context.Context, name string, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	hash := sha256.Sum256(data)
	r := api.Request{Action: "upload", FileName: name, Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
	cmd := c.Command(ctx, false, true, "exec "+AgentPath+" upload "+api.Encode(r))
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("upload failed: %w; run sess init %s if the remote helper is older than 0.7.0", ConnectionError(c.Host, stderr.String(), err), c.Host)
	}
	var res api.Response
	if err = json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("invalid upload response; run sess init %s", c.Host)
	}
	if res.Error != nil {
		return "", res.Error
	}
	if res.Protocol != api.Protocol || !strings.HasPrefix(res.Path, "/") || strings.ContainsAny(res.Path, "\r\n\x1b\x00") {
		return "", fmt.Errorf("invalid remote upload path")
	}
	return res.Path, nil
}
