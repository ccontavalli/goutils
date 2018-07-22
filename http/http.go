// Various functions to aid writing http servers.

package http

import (
	"strings"
	"net/http"
	"github.com/ccontavalli/goutils/misc"
	"golang.org/x/net/idna"
)

// Normalizes an hostname as received from a well behaving browser.
// Useful for later matching the string in a router.
func NormalizeRequestHostname(def string) string {
	return strings.ToLower(def)
}

// Normalizes an hostname as typed by a user. Specifically, it takes
// care of converting unicode characters in the equivalent punycode.
func NormalizeUserHostname(def string) (string, error) {
	result, err := idna.ToASCII(def)
	if err != nil {
		return def, err
	}
	return NormalizeRequestHostname(result), nil
}

// Extracts the hostname in a received HTTP request.
// If no hostname can be found, returns the supplied default.
// Does not perform any normalization.
func GetHost(r *http.Request, def string) string {
	if r.Host != "" {
		return misc.LooselyGetHost(r.Host)
	}
	if r.URL != nil && r.URL.Host != "" {
		return misc.LooselyGetHost(r.URL.Host)
	}
	return misc.LooselyGetHost(def)
}

// Lazily parse an "Accept-Encoding" header, returns true if the
// specified encoding is supported.
//
// For example, to check if a gzip compressed reply can be sent
// to a browser in response to an http request, use:
//
//    if AcceptsEncoding(request.Header.Get("Accept-Encoding"), "gzip") {
//      ...
//
func AcceptsEncoding(accepts, encoding string) bool {
	index := strings.Index(accepts, encoding)
	if index < 0 {
		return false
	}

	left := accepts[index+len(encoding):]
	if !strings.HasPrefix(left, ";q=0") {
		return true
	}
	left = left[len(";q=0"):]
	for i := 0; ; i++ {
		if i >= len(left) {
			return true
		}
		if left[i] == '.' {
			continue
		}

		if left[i] < '0' && left[i] > '9' {
			return true
		}
		if left[i] != '0' {
			return false
		}
	}

	// Never reached
	return false
}
