package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndHostFlag(t *testing.T) {
	t.Setenv("SESS_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	for _, args := range [][]string{{"--help"}, {"new", "--help"}, {"attach", "--help"}} {
		var b bytes.Buffer
		c := New(&b, &b)
		c.SetArgs(args)
		if e := c.Execute(); e != nil {
			t.Fatal(e)
		}
		s := b.String()
		if !strings.Contains(s, "--host") || !strings.Contains(s, "--help") {
			t.Fatal(s)
		}
	}
	var b bytes.Buffer
	c := New(&b, &b)
	c.SetArgs([]string{"set", "-h", "dev"})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(b.String(), "Default host: dev") {
		t.Fatal(b.String())
	}
}
func TestAliases(t *testing.T) {
	r := New(&bytes.Buffer{}, &bytes.Buffer{})
	for a, w := range map[string]string{"n": "new", "a": "attach", "rm": "remove"} {
		c, _, e := r.Find([]string{a})
		if e != nil || c.Name() != w {
			t.Fatal(a, e)
		}
	}
}
func TestNonTTYNewDoesNotMutate(t *testing.T) {
	t.Setenv("SESS_CONFIG", filepath.Join(t.TempDir(), "config"))
	r := New(&bytes.Buffer{}, &bytes.Buffer{})
	r.SetArgs([]string{"new", "work", "-h", "unreachable"})
	e := r.Execute()
	if e == nil || !strings.Contains(e.Error(), "--detach") {
		t.Fatal(e)
	}
}
