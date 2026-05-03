// Package httpserver is the canonical HTTP server runner. Every binary mode
// previously hand-rolled a near-identical Run/Shutdown dance; this package
// owns it once. Callers build a chi router (or anything implementing
// http.Handler) and call [Run].
package httpserver
