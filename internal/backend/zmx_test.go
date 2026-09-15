package backend

import (
	"context"
	"errors"
	"github.com/DeepakSilaych/sess/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseList(t *testing.T) {
	ss, e := ParseList("name=work\tpid=42\tclients=0\tcreated=1700000000\tcwd=/home/user/project with spaces\tsess_id=abc\nname=api\tpid=43\tclients=2\tcreated=1700000001\tcwd=/root\tsess_id=def\n")
	if e != nil || len(ss) != 2 {
		t.Fatal(ss, e)
	}
	if ss[0].Name != "api" || ss[0].Status != "attached" || ss[1].Directory != "/home/user/project with spaces" {
		t.Fatalf("%+v", ss)
	}
	if _, e = ParseList("name=work\terr=Timeout\tstatus=unreachable"); e == nil {
		t.Fatal("unreachable session treated as absent")
	}
	if _, e = ParseList("name=bad\tclients=no"); e == nil {
		t.Fatal("invalid backend contract accepted")
	}
}
func TestFindIdentity(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "zmx")
	if e := os.WriteFile(bin, []byte("#!/bin/sh\nprintf 'name=work\\tpid=42\\tclients=0\\tcreated=1700000000\\tsess_id=new\\n'\n"), 0700); e != nil {
		t.Fatal(e)
	}
	z := &ZMX{bin, d, d}
	_, e := z.Find(context.Background(), "work", "old")
	var ae *api.Error
	if !errors.As(e, &ae) || ae.Code != "replaced" {
		t.Fatal(e)
	}
	_, e = z.Find(context.Background(), "missing", "")
	if !errors.As(e, &ae) || ae.Code != "not_found" {
		t.Fatal(e)
	}
}
func TestPrivateRuntime(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "public")
	os.Mkdir(p, 0755)
	t.Setenv("SESS_RUNTIME_DIR", p)
	if _, e := Open(); e == nil {
		t.Fatal("accepted shared runtime")
	}
}
func TestEnvironmentIsolation(t *testing.T) {
	t.Setenv("ZMX_SESSION", "unrelated")
	t.Setenv("ZMX_DIR", "/other")
	z := &ZMX{Runtime: "/private", Home: "/home/test"}
	env := strings.Join(z.Env("api"), "\n")
	if strings.Contains(env, "ZMX_SESSION=") || strings.Contains(env, "ZMX_DIR=/other") {
		t.Fatal(env)
	}
	if !strings.Contains(env, "SESS_SESSION=api") {
		t.Fatal(env)
	}
}
