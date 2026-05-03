// Package aibot is the composition root for the `ai-bot` binary mode.
// The mode polls Postgres for matches whose slots have been flagged as
// AI-controlled (by the worker's inactivity scanner), then for each
// unattended slot opens a real WebSocket against the gateway as a
// service-account user. The session goroutine in session.go drives the
// same wire.Hello / Command / Resync protocol the React client uses;
// decisions come from the pure [heuristic] package.
//
// Architecture summary:
//
//	Postgres                    NATS                    HTTP
//	   |                          |                       |
//	   v                          v                       v
//	bot-pool          state.<match>.slot.<slot>      core-api/auth/login-as-bot
//	   |              (subscribed via gateway)            |
//	   v                                                  v
//	 session  -- WS --> gateway -- NATS --> engine -- state --> session
//	   |                                                  ^
//	   +------------------ commands -----------------------+
package aibot
