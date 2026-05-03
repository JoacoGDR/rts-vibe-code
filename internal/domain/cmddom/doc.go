// Package cmddom is the command catalogue. Each player order (move,
// recruit, construct, …future declare_war, propose_treaty, …) is a Go
// type implementing [Handler]; registry.go wires them up. Adding a new
// command is one new file plus one Register call — no switch-on-string
// dispatcher to grow.
//
// Commands are pure: they read a *matchdom.Match, mutate it, and return
// a (result, error) pair where result is a stable label suitable for
// metrics ("ok", "no_funds", "unauthorized", …).
package cmddom
