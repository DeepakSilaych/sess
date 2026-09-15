package api

import "testing"

func TestValidation(t *testing.T) {
	for _, s := range []string{"dev", "user@dev.example.com", "user@[2001:db8::1]"} {
		if e := ValidateHost(s); e != nil {
			t.Errorf("%s: %v", s, e)
		}
	}
	for _, s := range []string{"-oProxyCommand=id", "user:host", "dev;touch /tmp/x", "user@host\n", "$(id)", "dev host", "ssh://dev"} {
		if ValidateHost(s) == nil {
			t.Errorf("accepted unsafe host %q", s)
		}
	}
	for _, s := range []string{"../bad", "a/b", "-flag", "a;b", "name with spaces", ""} {
		if ValidateName(s) == nil {
			t.Errorf("accepted name %q", s)
		}
	}
}
func TestWireRoundTrip(t *testing.T) {
	r, e := Decode(Encode(Request{Action: "create", Name: "build-42"}))
	if e != nil || r.Name != "build-42" || r.Protocol != Protocol {
		t.Fatalf("%+v %v", r, e)
	}
	if _, e = Decode("e30"); e == nil {
		t.Fatal("accepted missing protocol")
	}
}
