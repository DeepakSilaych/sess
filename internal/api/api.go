// Package api defines the versioned SSH command contract shared by sess and its agent.
package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const Version = "0.6.0"
const Protocol = 1
const ZMXVersion = "0.8.1"

type Session struct {
	Name      string    `json:"name"`
	ID        string    `json:"id"`
	Clients   int       `json:"clients"`
	PID       int       `json:"pid"`
	Created   time.Time `json:"created"`
	Directory string    `json:"directory"`
	Status    string    `json:"status"`
}
type Request struct {
	Protocol int    `json:"protocol"`
	Action   string `json:"action"`
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty"`
}
type Response struct {
	Protocol int       `json:"protocol"`
	Version  string    `json:"version"`
	Backend  string    `json:"backend,omitempty"`
	Sessions []Session `json:"sessions,omitempty"`
	Session  *Session  `json:"session,omitempty"`
	Error    *Error    `json:"error,omitempty"`
}
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func (e *Error) Error() string {
	if e.Hint != "" {
		return e.Message + "\n\n" + e.Hint
	}
	return e.Message
}
func Fail(code, message, hint string) error { return &Error{code, message, hint} }

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,47}$`)

func ValidateName(s string) error {
	if !nameRE.MatchString(s) {
		return Fail("invalid_name", "Invalid session name: "+s, "Use 1–48 letters, numbers, hyphens or underscores; begin with a letter or number.")
	}
	return nil
}

// Only SSH destinations, never options, URIs, port suffixes or shell fragments.
var hostRE = regexp.MustCompile(`^(?:[a-zA-Z0-9_][a-zA-Z0-9_.-]*@)?(?:[a-zA-Z0-9_][a-zA-Z0-9_.-]*|\[[a-fA-F0-9:]+\])$`)

func ValidateHost(s string) error {
	if len(s) > 253 || !hostRE.MatchString(s) {
		return Fail("invalid_host", "Invalid SSH destination: "+s, "Use user@host or an SSH alias. Configure ports and keys in ~/.ssh/config.")
	}
	return nil
}
func Encode(r Request) string {
	r.Protocol = Protocol
	b, _ := json.Marshal(r)
	return base64.RawURLEncoding.EncodeToString(b)
}
func Decode(s string) (Request, error) {
	var r Request
	if len(s) > 4096 {
		return r, fmt.Errorf("request too large")
	}
	b, e := base64.RawURLEncoding.DecodeString(s)
	if e != nil {
		return r, e
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(&r); e != nil {
		return r, e
	}
	if r.Protocol != Protocol {
		return r, Fail("protocol", "Incompatible sess agent.", "Run sess init <host> to update the remote helper.")
	}
	return r, nil
}
