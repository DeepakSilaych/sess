package transport

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestSplitPaths(t *testing.T) {
	for _, s := range []string{`'/tmp/a b.png' /tmp/c.png `, `/tmp/a\ b.png "/tmp/c.png"`, `/tmp/a\ b.png /tmp/c.png`} {
		got, ok := splitPaths(s)
		if !ok || len(got) != 2 || got[0] != "/tmp/a b.png" || got[1] != "/tmp/c.png" {
			t.Fatalf("%q: %v %v", s, got, ok)
		}
	}
	for _, s := range []string{"explain /tmp/a.png", "/tmp/a.png\nrm -rf x", "'/tmp/a.png", "relative.png", ""} {
		if _, ok := splitPaths(s); ok {
			t.Fatalf("accepted %q", s)
		}
	}
}
func TestPasteFilter(t *testing.T) {
	input := "hello\x1b[A" + pasteStart + "/tmp/image.png" + pasteEnd + " world\r"
	for step := 1; step <= len(input); step++ {
		var out bytes.Buffer
		f := pasteFilter{write: func(b []byte) error { _, e := out.Write(b); return e }, transform: func(b []byte) ([]byte, error) {
			if string(b) != "/tmp/image.png" {
				t.Fatal(string(b))
			}
			return []byte("/remote/image.png"), nil
		}, notice: func(e error) { t.Fatal(e) }}
		for i := 0; i < len(input); i += step {
			if e := f.feed([]byte(input[i:min(i+step, len(input))])); e != nil {
				t.Fatal(e)
			}
		}
		if out.String() != "hello\x1b[A"+pasteStart+"/remote/image.png"+pasteEnd+" world\r" {
			t.Fatalf("chunk=%d got %q", step, out.String())
		}
	}
}
func TestPasteFailureAndBound(t *testing.T) {
	var out bytes.Buffer
	notices := 0
	f := pasteFilter{write: func(b []byte) error { _, e := out.Write(b); return e }, transform: func(b []byte) ([]byte, error) { return nil, errors.New("offline") }, notice: func(e error) { notices++ }}
	f.feed([]byte(pasteStart + "/tmp/a.png" + pasteEnd))
	if out.String() != pasteStart+pasteEnd || notices != 1 {
		t.Fatal("failed upload pasted local path")
	}
	out.Reset()
	f.feed([]byte("\x1b"))
	f.idle()
	if out.String() != "\x1b" {
		t.Fatal("escape swallowed")
	}
	out.Reset()
	large := strings.Repeat("x", maxPaste+4096)
	f.feed([]byte(pasteStart))
	for i := 0; i < len(large); i += 128 {
		f.feed([]byte(large[i:min(i+128, len(large))]))
	}
	f.feed([]byte(pasteEnd))
	if out.String() != pasteStart+large+pasteEnd || notices != 1 {
		t.Fatal("large paste altered")
	}
}
