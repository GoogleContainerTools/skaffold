package url

import (
	"regexp"
	"runtime"
	"strings"
)

var (
	isSchemeRegExp = regexp.MustCompile(`^[^:]+://`)

	// Ref: https://github.com/git/git/blob/v2.54.0/Documentation/urls.adoc#L41-L48
	scpLikeUrlRegExp = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:\s]+):(?:(?P<port>[0-9]{1,5}):)?(?P<path>[^\\].*)$`)
)

// MatchesScheme returns true if the given string matches a URL-like
// format scheme.
func MatchesScheme(url string) bool {
	return isSchemeRegExp.MatchString(url)
}

// MatchesScpLike returns true if the given string matches an SCP-like
// format scheme.
func MatchesScpLike(url string) bool {
	if !scpLikeUrlRegExp.MatchString(url) {
		return false
	}
	// Mirror canonical Git's url_is_local_not_ssh in connect.c[1] for
	// the cases the regex above cannot disambiguate by itself: a URL
	// is treated as a local path (not SCP-style SSH) when a `/`
	// precedes the first `:` (e.g. `./relative:path`,
	// `/abs/with:colon/file`), or — on Windows only — when it has a
	// DOS drive prefix like `C:foo` where the host is a single
	// ASCII letter.
	//
	// [1]: https://github.com/git/git/blob/v2.54.0/connect.c#L710-L716
	if before, _, _ := strings.Cut(url, ":"); strings.Contains(before, "/") {
		return false
	}
	if runtime.GOOS == "windows" && hasDosDrivePrefix(url) {
		return false
	}
	return true
}

// hasDosDrivePrefix reports whether s begins with `<letter>:` (a
// Windows drive prefix such as `C:` or `c:`). Mirrors canonical Git's
// win32_has_dos_drive_prefix[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/compat/win32/path-utils.c#L20-L29
func hasDosDrivePrefix(s string) bool {
	if len(s) < 2 || s[1] != ':' {
		return false
	}
	c := s[0]
	return ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')
}

// FindScpLikeComponents returns the user, host, port and path of the
// given SCP-like URL.
func FindScpLikeComponents(url string) (user, host, port, path string) {
	m := scpLikeUrlRegExp.FindStringSubmatch(url)
	return m[1], m[2], m[3], m[4]
}

// IsLocalEndpoint returns true if the given URL string specifies a
// local file endpoint.  For example, on a Linux machine,
// `/home/user/src/go-git` would match as a local endpoint, but
// `https://github.com/src-d/go-git` would not.
func IsLocalEndpoint(url string) bool {
	return !MatchesScheme(url) && !MatchesScpLike(url)
}
