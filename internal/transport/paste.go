package transport

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

const pasteStart = "\x1b[200~"
const pasteEnd = "\x1b[201~"
const maxPaste = 64 * 1024

// splitPaths decodes terminal shell quoting without evaluating shell syntax.
func splitPaths(text string) ([]string, bool) {
	var paths []string
	var word strings.Builder
	var quote rune
	escaped := false
	for _, r := range text {
		if r < 32 || r == 127 {
			return nil, false
		}
		if escaped {
			word.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if unicode.IsSpace(r) {
			if word.Len() > 0 {
				paths = append(paths, word.String())
				word.Reset()
			}
			continue
		}
		word.WriteRune(r)
	}
	if escaped || quote != 0 {
		return nil, false
	}
	if word.Len() > 0 {
		paths = append(paths, word.String())
	}
	if len(paths) == 0 || len(paths) > 8 {
		return nil, false
	}
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			return nil, false
		}
	}
	return paths, true
}

func (c *Client) imagePaste(ctx context.Context, text []byte) ([]byte, error) {
	paths, ok := splitPaths(string(text))
	if !ok {
		return text, nil
	}
	// Recognize only complete pastes consisting solely of existing local images.
	var files [][]byte
	total := 0
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" {
			return text, nil
		}
		data, err := readUpload(p, true)
		if err != nil {
			return text, nil
		}
		total += len(data)
		if total > 25*1024*1024 {
			return text, nil
		}
		files = append(files, data)
	}
	var remote []string
	for i, p := range paths {
		dst, err := c.uploadBytes(ctx, filepath.Base(p), files[i])
		if err != nil {
			return nil, fmt.Errorf("image upload failed (%s): %w", filepath.Base(p), err)
		}
		// Single quotes protect spaces and shell metacharacters, including when pasted at a shell prompt.
		remote = append(remote, "'"+strings.ReplaceAll(dst, "'", "'\\''")+"'")
	}
	return []byte(strings.Join(remote, " ") + " "), nil
}

// pasteFilter buffers only bracketed pastes; all ordinary input passes through.
// Oversized pastes are streamed unchanged with bounded memory.
type pasteFilter struct {
	pending   []byte
	pasting   bool
	streaming bool
	transform func([]byte) ([]byte, error)
	write     func([]byte) error
	notice    func(error)
}

func (p *pasteFilter) feed(data []byte) error {
	p.pending = append(p.pending, data...)
	for len(p.pending) > 0 {
		marker := []byte(pasteStart)
		if p.pasting {
			marker = []byte(pasteEnd)
		}
		if i := bytes.Index(p.pending, marker); i >= 0 {
			body := p.pending[:i]
			if p.pasting {
				if !p.streaming && len(body) <= maxPaste {
					var err error
					body, err = p.transform(body)
					if err != nil {
						p.notice(err)
						body = nil
					}
				}
				if !p.streaming {
					if err := p.write([]byte(pasteStart)); err != nil {
						return err
					}
				}
				if err := p.write(body); err != nil {
					return err
				}
				if err := p.write(marker); err != nil {
					return err
				}
				p.pasting = false
				p.streaming = false
			} else {
				if err := p.write(body); err != nil {
					return err
				}
				p.pasting = true
			}
			p.pending = p.pending[i+len(marker):]
			continue
		}
		if p.pasting && !p.streaming && len(p.pending) <= maxPaste {
			return nil
		}
		if p.pasting && !p.streaming {
			if err := p.write([]byte(pasteStart)); err != nil {
				return err
			}
			p.streaming = true
		}
		keep := 0
		for n := 1; n < len(marker) && n <= len(p.pending); n++ {
			if bytes.Equal(p.pending[len(p.pending)-n:], marker[:n]) {
				keep = n
			}
		}
		send := len(p.pending) - keep
		if err := p.write(p.pending[:send]); err != nil {
			return err
		}
		p.pending = append([]byte(nil), p.pending[send:]...)
		return nil
	}
	return nil
}

// Flush a lone escape promptly without timing out an in-progress bracketed paste.
func (p *pasteFilter) idle() error {
	if !p.pasting && len(p.pending) > 0 {
		err := p.write(p.pending)
		p.pending = nil
		return err
	}
	return nil
}
