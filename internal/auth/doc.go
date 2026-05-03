// Package auth contains the cross-cutting authentication primitives:
// JWT issuance and verification, password hashing, and the WebSocket
// ticket broker that bridges core-api sessions to gateway connections.
//
// auth lives outside the layered tree on purpose — it's used by both
// services (for issuing tokens) and adapters (for verifying them on
// requests). Treat it as a small "auth platform".
package auth
