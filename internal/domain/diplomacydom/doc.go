// Package diplomacydom owns the in-memory diplomatic state for a match:
// pairwise treaties (war / peace / alliance) and unilateral pacts
// (share-map and right-of-way grants). It is pure Go: no infra, no
// service imports. The engine carries one [Registry] per match and
// consults it on the hot path (combat resolution, visibility computation,
// movement validation).
//
// Treaty pairs are normalised so (red, blue) and (blue, red) resolve to
// the same record. Pacts are directional and keyed by (granter, grantee).
package diplomacydom
