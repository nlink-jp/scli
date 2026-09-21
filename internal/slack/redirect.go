package slack

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// tokenMayFollowRedirect reports whether the Slack token may be re-sent to
// target when the download started at origin.
//
// Why the token is re-sent at all: Slack's url_private_download answers with a
// redirect, and Go's http.Client re-sends Authorization only to the same host
// or a subdomain of it (isDomainOrSubdomain), so a redirect to a sibling host
// arrives unauthenticated. v1.7.1 added the re-attachment against a target
// that answered an unauthenticated request with an HTML sign-in page at status
// 200 — a "downloaded" file that is a web page. Note that v1.7.2 then found a
// missing files:read scope producing the same page, so whether the target
// still requires the token has not been re-measured. It is kept because it
// costs nothing if unnecessary and a download silently returning a web page is
// expensive; see downloadFileTo and the regression tests named there.
//
// Why it is scoped: the previous rule re-attached the token to whatever host
// the redirect named, for ten hops. A redirect is chosen by the server, so
// that handed the workspace token to any host Slack — or anything answering
// for it — pointed at. The token now travels only within the domain the
// download started in, which is the property that makes re-attaching safe
// rather than merely convenient.
//
// Both hosts are judged against via[0], the URL the caller asked for, so a
// chain cannot walk the token out one hop at a time.
func tokenMayFollowRedirect(origin, target *url.URL) bool {
	if origin == nil || target == nil {
		return false
	}
	// Never downgrade: a token sent in cleartext is a leaked token.
	if origin.Scheme == "https" && target.Scheme != "https" {
		return false
	}
	oh, th := origin.Hostname(), target.Hostname()
	if oh == "" || th == "" {
		return false
	}
	if strings.EqualFold(oh, th) {
		return true
	}
	od, td := registrableDomain(oh), registrableDomain(th)
	return od != "" && strings.EqualFold(od, td)
}

// registrableDomain returns the domain that owns host, as far as that can be
// told without a public-suffix list. It is an approximation, and it is built
// to err towards *narrower* than the truth: a domain read too narrowly only
// withholds the token (the download then fails, and says why), while one read
// too widely would hand it to a stranger.
//
// The last two labels are the answer for a host under a single-label suffix
// (files.slack.com -> slack.com). Under a two-part suffix the same rule would
// return co.uk and make every .co.uk host one domain, so a two-letter final
// label after a short one takes a third label (co.uk, co.jp, com.au). An IP
// literal owns nothing but itself, and is returned unchanged so that only an
// exact match can satisfy the caller.
//
// Slack serves file downloads from slack.com, for which this is exact. If a
// redirect is ever measured leaving that domain, replace this with the list of
// hosts actually observed rather than widening the rule.
func registrableDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return ""
	}
	if net.ParseIP(host) != nil {
		return host
	}
	labels := strings.Split(host, ".")
	want := 2
	if n := len(labels); n >= 3 && len(labels[n-1]) == 2 && len(labels[n-2]) <= 3 {
		want = 3
	}
	if len(labels) <= want {
		return host
	}
	return strings.Join(labels[len(labels)-want:], ".")
}

// withheldNote describes the hosts a redirect reached without the token, for
// an error message. A download that silently produced a sign-in page was the
// original bug here; a download that fails because the token was held back
// has to name the host, or the next reader cannot tell the two apart.
func withheldNote(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}
	return fmt.Sprintf(" (the Slack token was not sent to %s: the redirect left the domain the download started in)",
		strings.Join(hosts, ", "))
}
