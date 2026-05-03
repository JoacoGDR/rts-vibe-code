// Package chatsvc owns the use-case orchestration for in-match chat:
// scope/body validation, throttling, persistence and live fan-out. The
// gateway is responsible for the upstream membership decision (it knows
// the caller's slot and the live diplomacy state); chatsvc enforces the
// rules that do not require simulation context.
package chatsvc
