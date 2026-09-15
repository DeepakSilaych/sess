package backend

import (
	"context"
	"os"
	"testing"
)

// Optional native-backend check, in addition to the real SSH Docker suite.
func TestNativeZMX(t *testing.T) {
	bin := os.Getenv("SESS_TEST_ZMX")
	if bin == "" {
		t.Skip("set SESS_TEST_ZMX to a verified zmx 0.8.1 executable")
	}
	dir, e := os.MkdirTemp("/tmp", "sess-native-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	t.Setenv("HOME", dir)
	t.Setenv("SHELL", "/bin/sh")
	z := &ZMX{Binary: bin, Runtime: dir, Home: dir}
	ctx := context.Background()
	if _, e = z.Version(ctx); e != nil {
		t.Fatal(e)
	}
	s, e := z.Create(ctx, "native-test")
	if e != nil {
		t.Fatal(e)
	}
	defer z.Remove(ctx, s.Name, s.ID)
	if s.ID == "" || s.Clients != 0 {
		t.Fatalf("unexpected created state: %+v", s)
	}
	if _, e = z.Create(ctx, "native-test"); e == nil {
		t.Fatal("duplicate creation succeeded")
	}
	if _, e = z.Find(ctx, s.Name, "wrong-identity"); e == nil {
		t.Fatal("reattached a replacement")
	}
	if e = z.Remove(ctx, s.Name, s.ID); e != nil {
		t.Fatal(e)
	}
	if ss, e := z.List(ctx); e != nil || len(ss) != 0 {
		t.Fatal(ss, e)
	}
}
