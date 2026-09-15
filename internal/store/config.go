package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"os"
	"path/filepath"
)

type Config struct {
	Version int      `json:"version"`
	Host    string   `json:"host"`
	Hosts   []string `json:"hosts,omitempty"`
}

func Path() (string, error) {
	if p := os.Getenv("SESS_CONFIG"); p != "" {
		return p, nil
	}
	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		h, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		d = filepath.Join(h, ".config")
	}
	return filepath.Join(d, "sess", "config.json"), nil
}
func Load() (Config, error) {
	c := Config{Version: 1}
	p, e := Path()
	if e != nil {
		return c, e
	}
	b, e := os.ReadFile(p)
	if errors.Is(e, os.ErrNotExist) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	if e = json.Unmarshal(b, &c); e != nil {
		return c, fmt.Errorf("read %s: %w", p, e)
	}
	if c.Version != 1 {
		return c, fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Host != "" {
		e = api.ValidateHost(c.Host)
	}
	return c, e
}
func Save(c Config) error {
	p, e := Path()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".config-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(append(b, '\n')); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), p)
}
func SetHost(host string) error {
	if e := api.ValidateHost(host); e != nil {
		return e
	}
	c, e := Load()
	if e != nil {
		return e
	}
	c.Host = host
	for _, h := range c.Hosts {
		if h == host {
			return Save(c)
		}
	}
	c.Hosts = append(c.Hosts, host)
	return Save(c)
}
func Resolve(override string) (string, error) {
	if override != "" {
		return override, api.ValidateHost(override)
	}
	c, e := Load()
	if e != nil {
		return "", e
	}
	if c.Host == "" {
		return "", api.Fail("no_host", "No default host configured.", "Run sess init <host>, then sess set --host <host>. Or pass --host for this command.")
	}
	return c.Host, nil
}
