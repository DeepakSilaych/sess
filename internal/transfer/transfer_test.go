package transfer

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReceive(t *testing.T) {
	home := t.TempDir()
	data := []byte("image bytes")
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	one, e := Receive(home, "test image.png", int64(len(data)), sum, bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	two, e := Receive(home, "test image.png", int64(len(data)), sum, bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if one == two {
		t.Fatal("overwrote prior upload")
	}
	b, _ := os.ReadFile(one)
	if !bytes.Equal(b, data) {
		t.Fatal("changed bytes")
	}
	st, _ := os.Stat(one)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	for _, tc := range []struct {
		name string
		size int64
		sum  string
		data []byte
	}{
		{"../escape", 1, sum, data}, {"bad\nname", 1, sum, data}, {"a", MaxSize + 1, sum, data},
		{"a", int64(len(data)), sum, data[:2]}, {"a", 2, sum, data}, {"a", int64(len(data)), fmt.Sprintf("%064d", 0), data},
	} {
		if _, e := Receive(home, tc.name, tc.size, tc.sum, bytes.NewReader(tc.data)); e == nil {
			t.Fatalf("accepted invalid upload: %+v", tc)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(filepath.Dir(one)))
	if len(entries) != 2 {
		t.Fatalf("partial uploads left behind: %d", len(entries))
	}
}
