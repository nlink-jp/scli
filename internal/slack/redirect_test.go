package slack

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return u
}

// TestTokenMayFollowRedirect pins the rule that decides where the workspace
// token may travel. The token is re-attached because Slack's download target
// refuses to serve the bytes without it (see downloadFileTo); the rule is what
// keeps that from meaning "to anywhere the redirect names".
func TestTokenMayFollowRedirect(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		target string
		want   bool
	}{
		{"same host", "https://files.slack.com/files-pri/F1", "https://files.slack.com/x", true},
		{"subdomain of origin", "https://slack.com/f", "https://files.slack.com/x", true},
		{"sibling host, same domain", "https://files.slack.com/f", "https://cdn.slack.com/x", true},
		{"host differs only in case", "https://files.slack.com/f", "https://FILES.Slack.com/x", true},
		{"port differs", "https://files.slack.com/f", "https://files.slack.com:8443/x", true},
		{"apex from subdomain", "https://files.slack.com/f", "https://slack.com/x", true},

		// slack-files.com is Slack's, but it is a different registrable
		// domain, so the rule holds the token back. If a real download is
		// ever observed redirecting there, add the measured host — do not
		// widen the rule to "names containing slack".
		{"different Slack-owned domain", "https://files.slack.com/f", "https://slack-files.com/x", false},

		{"unrelated host", "https://files.slack.com/f", "https://evil.example/x", false},
		{"lookalike suffix", "https://files.slack.com/f", "https://files.slack.com.evil.example/x", false},
		{"downgrade to http", "https://files.slack.com/f", "http://files.slack.com/x", false},
		{"different IP literal", "http://127.0.0.1/f", "http://10.0.0.1/x", false},
		{"same IP literal", "http://127.0.0.1/f", "http://127.0.0.1:9/x", true},
		{"IP to name", "http://127.0.0.1/f", "http://localhost/x", false},

		// Both are under the co.uk suffix; a two-label reading would call
		// them one domain and hand the token across two organisations.
		{"two-part suffix, different owners", "https://a.example.co.uk/f", "https://b.other.co.uk/x", false},
		{"two-part suffix, same owner", "https://a.example.co.uk/f", "https://b.example.co.uk/x", true},

		{"no host", "https://files.slack.com/f", "/relative", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tokenMayFollowRedirect(mustURL(t, tc.origin), mustURL(t, tc.target))
			if got != tc.want {
				t.Errorf("tokenMayFollowRedirect(%s, %s) = %v, want %v", tc.origin, tc.target, got, tc.want)
			}
		})
	}
	if tokenMayFollowRedirect(nil, mustURL(t, "https://files.slack.com/x")) {
		t.Error("a nil origin must not authorise the token")
	}
}

func TestRegistrableDomain(t *testing.T) {
	cases := map[string]string{
		"files.slack.com":    "slack.com",
		"a.b.c.slack.com":    "slack.com",
		"slack.com":          "slack.com",
		"com":                "com",
		"FILES.SLACK.COM":    "slack.com",
		"files.slack.com.":   "slack.com",
		"a.example.co.uk":    "example.co.uk",
		"a.example.co.jp":    "example.co.jp",
		"a.b.example.com.au": "example.com.au",
		"127.0.0.1":          "127.0.0.1",
		"::1":                "::1",
		"localhost":          "localhost",
		"":                   "",
	}
	for host, want := range cases {
		if got := registrableDomain(host); got != want {
			t.Errorf("registrableDomain(%q) = %q, want %q", host, got, want)
		}
	}
}

// redirectingFileServer serves a Slack-shaped download: the private URL
// redirects to a second host, which answers with the file only when the
// request carries the token, and with a sign-in page otherwise — the real
// behaviour that makes re-attaching the header necessary.
//
// Every hostname is dialled to this one listener, so a redirect to a sibling
// host is a real cross-host redirect as far as http.Client is concerned (it
// strips Authorization) while still being reachable in a test without DNS.
func redirectingFileServer(t *testing.T, cdnHost string, content []byte) (*Client, *[]string) {
	t.Helper()

	var seenAuth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files-pri/F1":
			http.Redirect(w, r, "http://"+cdnHost+"/cdn/F1", http.StatusFound)
		case "/cdn/F1":
			seenAuth = append(seenAuth, r.Header.Get("Authorization"))
			if r.Header.Get("Authorization") == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte("<html><body>Sign in to Slack</body></html>"))
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	addr := srv.Listener.Addr().String()
	hc := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
	return &Client{token: "test-token", httpClient: hc, baseURL: srv.URL}, &seenAuth
}

// TestDownloadFileTo_SiblingHostRedirectGetsTheToken is the reason the
// re-attachment exists: Go re-sends Authorization to the same host or a
// subdomain, but not to a sibling, and Slack's target answers an
// unauthenticated request with a sign-in page. Remove the re-attachment and
// this test saves that page as the file.
func TestDownloadFileTo_SiblingHostRedirectGetsTheToken(t *testing.T) {
	content := []byte("the actual file bytes")
	c, seenAuth := redirectingFileServer(t, "cdn.slack.com", content)

	dest := filepath.Join(t.TempDir(), "F1_test.txt")
	if err := c.downloadFileTo(context.Background(), "http://files.slack.com/files-pri/F1", dest); err != nil {
		t.Fatalf("download across a sibling Slack host failed: %v", err)
	}

	got, err := os.ReadFile(dest) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("downloaded content = %q, want %q", got, content)
	}
	if len(*seenAuth) != 1 || (*seenAuth)[0] != "Bearer test-token" {
		t.Errorf("the redirect target saw Authorization %q, want the bearer token once", *seenAuth)
	}
}

// TestDownloadFileTo_ForeignHostRedirectIsRefused is the other half: the
// redirect target is the server's choice, so a hop out of Slack's domain must
// not carry the token. The download then fails — and has to say why, naming
// the host, or the next reader cannot tell this apart from an outage.
func TestDownloadFileTo_ForeignHostRedirectIsRefused(t *testing.T) {
	c, seenAuth := redirectingFileServer(t, "attacker.example", []byte("never served"))

	dest := filepath.Join(t.TempDir(), "F1_test.txt")
	err := c.downloadFileTo(context.Background(), "http://files.slack.com/files-pri/F1", dest)
	if err == nil {
		t.Fatal("a redirect out of the starting domain must fail the download, not save the page it returns")
	}
	if !strings.Contains(err.Error(), "attacker.example") {
		t.Errorf("error does not name the host the token was held back from: %v", err)
	}
	if len(*seenAuth) != 1 || (*seenAuth)[0] != "" {
		t.Errorf("the foreign host saw Authorization %q, want it absent", *seenAuth)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("a refused download must leave no file behind: stat %s = %v", dest, statErr)
	}
}
