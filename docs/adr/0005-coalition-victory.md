# ADR-0005: Coalition Victory and Coalition-Aware Combat

- **Status**: accepted
- **Date**: 2026-05-02
- **Deciders**: @joaquing
- **Phase**: Phase 4 (Diplomacy & Chat)

## Context

Phase 4 introduces the alliance stance. With alliances on the board the
existing combat and victory model breaks in two ways:

1. **Friendly fire.** Combat resolution stacked stationary units by
   `OwnerSlot` and then iterated until one slot stood. Two allied slots
   garrisoning the same province would happily kill each other.
2. **Premature victory.** `checkVictory` ended the match when only one
   slot held a capital. With alliances, a coalition of red+blue holding
   only red's capital should be considered victorious as a coalition,
   not "red won".

We considered three escape hatches: opt-in friendly fire, sticky-only
alliances (no in-province pooling), and full coalition rewrite. The
first two break the alliance fantasy; the third is what the rest of the
diplomacy code already assumes (visibility unions, right-of-way grants,
shared-map vision).

## Decision

The diplomacy registry (`internal/domain/diplomacydom`) computes
**coalitions transitively**: two slots are in the same coalition iff
there is a path of `Alliance` treaties between them. The deterministic
**coalition leader** is the alphabetically smallest slot in the
coalition. Both pieces of state are pure functions of the registry —
nothing is precomputed.

Combat changes (`internal/domain/matchdom/combat.go`):

- `collapseChampions` keys champions by `Diplomacy.CoalitionID(...)`
  instead of `OwnerSlot`. Allied unit HP is pooled into a single
  champion. The champion's `OwnerSlot` is the strongest contributor's
  slot so capture / display logic still has a sensible owner.
- `captureIfOwnerChanged` skips ownership transfer when the prior owner
  and the new champion belong to the same coalition (allies don't steal
  provinces from each other on garrison swaps).

Victory changes (`internal/domain/matchdom/victory.go`):

- `checkVictory` collects every slot still holding a capital, then
  groups them by coalition. The match ends only when **exactly one**
  coalition has any capital-holders left.
- The `match_ended` AppliedEvent gains an `extra` map with `coalition`
  (leader id) and `winners` (sorted slot list). `WinnerSlot` keeps its
  legacy contract by storing the coalition leader; `WinnerCoal` is the
  full slice for wire consumers.

Wire snapshot (`pkg/shared/wire/messages.go`) gains `Diplomacy
[]TreatyState` and `Pacts []PactState` (filtered for the viewer) so the
frontend can render stance grids without a separate diplomacy round
trip. The match-state filter in `internal/domain/visibility` always
includes diplomacy and only includes pacts where the viewer is granter
or grantee.

## Consequences

- **Positive**:
  - Allies can stack garrisons safely; this is what every Supremacy
    player expects.
  - Victory genuinely matches the in-game alliance situation; no more
    "your ally won, you didn't" surprises.
  - The coalition primitive is reusable — chat and the upcoming
    notification system (Phase 5) key off it for free.
- **Negative**:
  - Combat resolution is now O(units × slots) per province instead of
    O(units) — coalition lookup walks the alliance graph each time.
    Given ≤8 slots this is irrelevant; document for future load
    testing.
  - The `WinnerSlot` field now carries the coalition leader id, which
    may not be a real "winning player" if the leader was eliminated
    earlier. `WinnerCoal` is the canonical answer and consumers should
    prefer it.
- **Follow-ups**:
  - Phase 5 will reuse `CoalitionID` for system notifications scoped to
    a coalition.
  - Phase 7 should benchmark `checkVictory` once we have larger maps
    (≥ 12 slots) — the slot-iteration grouping is fine for MVP sizes.

## Alternatives considered

- **Keep slot-keyed combat, add an opt-in "no friendly fire" toggle
  per match.** Rejected — accidental friendly fire is the kind of
  bug players will quote on day one. The default has to be safe.
- **Sticky alliances only (no HP pooling).** Rejected — the
  visibility code already pools allied vision, so combat would feel
  arbitrarily stricter than every other interaction.
- **Cache coalitions on the registry.** Rejected — coalitions change
  every time a treaty changes. Cache invalidation is more code than
  the linear walk.
