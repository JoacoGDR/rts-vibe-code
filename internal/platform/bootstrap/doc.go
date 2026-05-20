// Package bootstrap is shared process startup for every supremacy binary:
// config load, logging, metrics HTTP server, signal handling, and graceful
// shutdown coordination with the mode-specific [Runner].
package bootstrap
