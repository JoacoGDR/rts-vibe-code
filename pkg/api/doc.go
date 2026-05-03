// Package api defines the public REST DTOs the HTTP API exchanges with
// clients. Keep these types stable: every shape change is an external
// contract change. Internal services translate to/from these via small
// mapper functions in the controller layer.
package api
