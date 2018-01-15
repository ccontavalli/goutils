// Various functions to aid writing http servers.

package http

import (
	"strings"
)

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
