package cluster

import (
	"testing"
	"time"
)

func TestChildProcessRuns(t *testing.T) {
	cmd := []string{"/opt/homebrew/bin/pinniped", "login", "oidc", "--issuer=x"}
	for _, tt := range []struct {
		name string
		c    childProcess
		want bool
	}{
		{"as started", childProcess{argv: []string{"/opt/homebrew/bin/pinniped", "login", "oidc", "--issuer=x"}}, true},
		{"a bare name", childProcess{argv: []string{"pinniped", "login", "oidc", "--issuer=x"}}, true},
		{"a script", childProcess{argv: []string{"/bin/sh", "/opt/homebrew/bin/pinniped", "login", "oidc", "--issuer=x"}}, true},
		{"other arguments", childProcess{argv: []string{"/opt/homebrew/bin/pinniped", "login", "oidc", "--issuer=y"}}, false},
		{"another program", childProcess{argv: []string{"/usr/bin/kubelogin", "login", "oidc", "--issuer=x"}}, false},
		{"the same name elsewhere", childProcess{argv: []string{"/usr/local/bin/pinniped", "login", "oidc", "--issuer=x"}}, false},
		{"a script elsewhere", childProcess{argv: []string{"/bin/sh", "/tmp/pinniped", "login", "oidc", "--issuer=x"}}, false},
		{"too short", childProcess{argv: []string{"login", "oidc", "--issuer=x"}}, false},
		{"Windows", childProcess{exe: "PINNIPED.EXE"}, true},
		{"Windows, another", childProcess{exe: "kubelogin.exe"}, false},
	} {
		if got := tt.c.runs(cmd); got != tt.want {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
	if (childProcess{argv: []string{"x"}}).runs(nil) {
		t.Error("no command matched a process")
	}
}

func TestLoginEndedErrorText(t *testing.T) {
	for _, tt := range []struct {
		e    *LoginEndedError
		want string
	}{
		{&LoginEndedError{Program: "pinniped", Bound: 5 * time.Minute, Cause: ErrLoginTimedOut}, "the login through pinniped did not finish within 5 minutes"},
		{&LoginEndedError{Program: "pinniped", Bound: time.Minute, Cause: ErrLoginTimedOut}, "the login through pinniped did not finish within 1 minute"},
		{&LoginEndedError{Program: "pinniped", Bound: 1500 * time.Millisecond, Cause: ErrLoginTimedOut}, "the login through pinniped did not finish within 1.5s"},
		{&LoginEndedError{Program: "kubelogin", Bound: 5 * time.Minute, Cause: ErrLoginCancelled}, "login cancelled: Cogpit ended kubelogin"},
	} {
		if got := tt.e.Error(); got != tt.want {
			t.Errorf("%q, want %q", got, tt.want)
		}
	}
}
