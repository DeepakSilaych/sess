package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostResolution(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SESS_CONFIG", p)
	if _, e := Resolve(""); e == nil {
		t.Fatal("missing host silently accepted")
	}
	if e := SetHost("dev"); e != nil {
		t.Fatal(e)
	}
	if h, e := Resolve("staging"); e != nil || h != "staging" {
		t.Fatal(h, e)
	}
	if h, _ := Resolve(""); h != "dev" {
		t.Fatal("override changed default")
	}
	info, e := os.Stat(p)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("config must be private", e)
	}
	if e = SetHost("dev"); e != nil {
		t.Fatal(e)
	}
	c, _ := Load()
	if len(c.Hosts) != 1 {
		t.Fatal("duplicate hosts")
	}
	if e = os.WriteFile(p, []byte("garbage"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Resolve(""); e == nil {
		t.Fatal("corrupt config ignored")
	}
}
