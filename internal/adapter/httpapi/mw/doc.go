// Package mw collects the HTTP middleware used by the public API: CORS,
// request logging, Prometheus instrumentation, and authentication. Each
// middleware is a single small file so adding/removing one is a single
// import change.
package mw
