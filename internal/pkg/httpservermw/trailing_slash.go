package httpservermw

import (
	"net/http"
	"strings"
)

// RemoveTrailingSlash is a middleware that remove trailing slashed
func RemoveTrailingSlash(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if req != nil && req.URL != nil {
			req.URL.Path = strings.TrimRight(req.URL.Path, "/")
		}

		next.ServeHTTP(w, req)
	}

	return http.HandlerFunc(fn)
}
