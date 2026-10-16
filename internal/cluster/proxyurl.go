package cluster

import (
	"cmp"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// A kubeconfig's own proxy-url (clusters[].cluster.proxy-url) may carry a user and password, as
// "http://user:password@proxy:3128". client-go honours it by itself (clientcmd sets
// rest.Config.Proxy, which Options.Proxy replaces), but its validation quotes a proxy-url it
// cannot use whole: `invalid 'proxy-url' "ftp://user:password@proxy" for cluster "c": ...`, and
// "could not parse: <the URL>". Connect therefore removes the user information of every proxy-url
// of the kubeconfig from the errors it returns (TestKubeconfigProxyPasswordNeverInErrors).

// minSecretLen is the shortest password replaced on its own: a shorter one would blot out ordinary
// words of the message, and the user information as a whole is replaced anyway.
const minSecretLen = 4

// kubeconfigProxySecrets returns the texts that would give away the user information of the
// kubeconfig's proxy URLs: the user information as written (everything between "://" and the last
// "@", so an unparseable URL is covered too), the password alone, and each as %q writes it.
func kubeconfigProxySecrets(cfg clientcmdapi.Config) []string {
	var out []string
	add := func(s string) {
		if s == "" {
			return
		}
		quoted := strconv.Quote(s)
		for _, v := range []string{s, quoted[1 : len(quoted)-1]} {
			if !slices.Contains(out, v) {
				out = append(out, v)
			}
		}
	}
	for _, cl := range cfg.Clusters {
		if cl == nil || !strings.Contains(cl.ProxyURL, "@") {
			continue
		}
		raw := cl.ProxyURL
		at := strings.LastIndex(raw, "@")
		start := 0
		if i := strings.Index(raw, "://"); i >= 0 && i < at {
			start = i + len("://")
		}
		info := raw[start:at]
		add(info)
		if _, pw, ok := strings.Cut(info, ":"); ok && len(pw) >= minSecretLen {
			add(pw)
			if unescaped, err := url.PathUnescape(pw); err == nil && len(unescaped) >= minSecretLen {
				add(unescaped)
			}
		}
	}
	// The longest first, so a password inside the user information does not split it.
	slices.SortStableFunc(out, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return out
}

// scrubSecrets replaces each secret in err's text by "•••", keeping a *ConnectError a
// *ConnectError. An error without any of them is returned as it is.
func scrubSecrets(err error, secrets []string) error {
	if err == nil || len(secrets) == 0 {
		return err
	}
	scrub := func(s string) string { return replaceSecrets(s, secrets) }
	var ce *ConnectError
	if errors.As(err, &ce) {
		out := *ce
		out.Problem, out.Context = scrub(ce.Problem), scrub(ce.Context)
		if out.Problem == ce.Problem && out.Context == ce.Context {
			return err
		}
		return &out
	}
	if text := scrub(err.Error()); text != err.Error() {
		return errors.New(text)
	}
	return err
}

// replaceSecrets replaces each of secrets in s by "•••", the longest first, so that a secret inside
// a longer one does not split it. Empty secrets are skipped.
func replaceSecrets(s string, secrets []string) string {
	if len(secrets) == 0 {
		return s
	}
	sorted := slices.Clone(secrets)
	slices.SortStableFunc(sorted, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	for _, secret := range sorted {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "•••")
		}
	}
	return s
}
