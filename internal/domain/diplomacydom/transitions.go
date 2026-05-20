package diplomacydom

import "time"

// DeclareWar unilaterally moves the pair to War. Always allowed (modulo
// cooldown) because severing peace is intentionally cheap to initiate
// even if costly in-game. Cancels any pending offer between the pair and
// silently revokes any pacts in either direction.
func (r *Registry) DeclareWar(from, target string, now time.Time) error {
	if from == target {
		return ErrSelfTreaty
	}
	if r.Stance(from, target) == War {
		return ErrAlreadyAtStance
	}
	if t := r.Treaty(from, target); t != nil && now.Sub(t.ChangedAt) < CooldownBetweenChanges {
		return ErrCooldown
	}
	r.upsert(from, target, War, PendingNone, "", now, from)
	r.revokeAllBetween(from, target)
	r.bumpVersion()
	return nil
}

// ProposePeace marks a pending Peace offer from -> target. The other
// side accepts via AcceptPeace.
func (r *Registry) ProposePeace(from, target string, now time.Time) error {
	return r.propose(from, target, PendingPeace, now)
}

// ProposeAlliance marks a pending Alliance offer from -> target. Like
// peace, requires the recipient to accept.
func (r *Registry) ProposeAlliance(from, target string, now time.Time) error {
	return r.propose(from, target, PendingAlliance, now)
}

// AcceptPeace upgrades a pending PendingPeace into an actual Peace
// treaty. The accepting slot must be the recipient of the offer (i.e. not
// the proposer).
func (r *Registry) AcceptPeace(accepter, other string, now time.Time) error {
	return r.accept(accepter, other, PendingPeace, Peace, now)
}

// AcceptAlliance upgrades a pending PendingAlliance offer.
func (r *Registry) AcceptAlliance(accepter, other string, now time.Time) error {
	return r.accept(accepter, other, PendingAlliance, Alliance, now)
}

// GrantPact records a unilateral pact from -> to. Idempotent: re-granting
// an existing pact returns ErrPactAlreadyGiven so the command layer can
// surface it to the user.
func (r *Registry) GrantPact(from, to string, kind PactKind, now time.Time) error {
	if from == to {
		return ErrSelfTreaty
	}
	k := pactKey{From: from, To: to, Kind: kind}
	if _, ok := r.pacts[k]; ok {
		return ErrPactAlreadyGiven
	}
	r.pacts[k] = &Pact{From: from, To: to, Kind: kind, GrantedAt: now}
	r.bumpVersion()
	return nil
}

// RevokePact removes a previously-granted pact. Errors if the pact does
// not exist.
func (r *Registry) RevokePact(from, to string, kind PactKind) error {
	if from == to {
		return ErrSelfTreaty
	}
	k := pactKey{From: from, To: to, Kind: kind}
	if _, ok := r.pacts[k]; !ok {
		return ErrPactMissing
	}
	delete(r.pacts, k)
	r.bumpVersion()
	return nil
}

// propose centralises the proposal flow used by ProposePeace and
// ProposeAlliance.
func (r *Registry) propose(from, target string, kind Pending, now time.Time) error {
	if from == target {
		return ErrSelfTreaty
	}
	target1, target2 := sortedPair(from, target)
	t, ok := r.treaties[pairKey{target1, target2}]
	switch {
	case ok && t.Stance == requestedStance(kind):
		return ErrAlreadyAtStance
	case ok && now.Sub(t.ChangedAt) < CooldownBetweenChanges:
		return ErrCooldown
	}
	r.upsert(from, target, currentStance(t), kind, from, now, "")
	return nil
}

// accept consolidates the AcceptPeace / AcceptAlliance code paths.
func (r *Registry) accept(accepter, other string, want Pending, becomes Stance, now time.Time) error {
	t := r.Treaty(accepter, other)
	if t == nil || t.Pending != want || t.PendingFrom == accepter || t.PendingFrom != other {
		return ErrNoPendingOffer
	}
	r.upsert(accepter, other, becomes, PendingNone, "", now, accepter)
	return nil
}

// upsert writes a treaty record, normalising slot order. Helper used by
// every transition path so the sort-and-store dance lives in one place.
func (r *Registry) upsert(a, b string, stance Stance, pending Pending, pendingFrom string, now time.Time, changeFrom string) {
	x, y := sortedPair(a, b)
	t, ok := r.treaties[pairKey{x, y}]
	if !ok {
		t = &Treaty{SlotA: x, SlotB: y}
		r.treaties[pairKey{x, y}] = t
	}
	t.Stance = stance
	t.Pending = pending
	t.PendingFrom = pendingFrom
	t.ChangedAt = now
	if changeFrom != "" {
		t.LastChangeFrom = changeFrom
	}
	r.bumpVersion()
}

// revokeAllBetween clears every directional pact in either direction
// between the two slots. Called when war breaks out so share-map and
// right-of-way grants do not silently survive a hostile turn.
func (r *Registry) revokeAllBetween(a, b string) {
	changed := false
	for k := range r.pacts {
		if (k.From == a && k.To == b) || (k.From == b && k.To == a) {
			delete(r.pacts, k)
			changed = true
		}
	}
	if changed {
		r.bumpVersion()
	}
}

func currentStance(t *Treaty) Stance {
	if t == nil {
		return War
	}
	return t.Stance
}

func requestedStance(p Pending) Stance {
	switch p {
	case PendingPeace:
		return Peace
	case PendingAlliance:
		return Alliance
	default:
		return War
	}
}
