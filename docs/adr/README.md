# Architectural Decision Records

This folder captures the architectural decisions that shape the codebase.
Every non-trivial change to the layering rules, an external dependency,
or a major pattern should land an ADR alongside the code.

## Index

- [ADR-0001 — Layered Modular Monolith with Hex-Flavoured Adapters](0001-layered-modular-monolith.md)
- [ADR-0003 — Chat Routing](0003-chat-routing.md)
- [ADR-0004 — AI Bot as a Headless WebSocket Client](0004-ai-as-headless-client.md)
- [ADR-0005 — Coalition Victory](0005-coalition-victory.md)

## Authoring

Copy [`template.md`](template.md) to `NNNN-short-slug.md`, where `NNNN`
is the next four-digit number. Submit it with the PR that lands the
decision. Mark superseded ADRs explicitly: edit the predecessor's status
to point at the new ADR.
