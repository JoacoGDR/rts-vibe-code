package matchdom

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
	"github.com/joaquing/clone-supremacy/internal/domain/unitdom"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// New instantiates a Match from a map definition and slot -> userID
// assignments. Each starting unit is materialised at its province's coords.
func New(matchID, mapID string, slotAssignments map[string]string, speed float64, startedAt time.Time) (*Match, error) {
	m, err := maps.Load(mapID)
	if err != nil {
		return nil, err
	}

	match := &Match{
		ID:          matchID,
		MapID:       mapID,
		Map:         m,
		StartedAt:   startedAt,
		GameStart:   startedAt,
		GameNow:     startedAt,
		Speed:       speed,
		Players:     map[string]*Player{},
		Provinces:   map[string]*Province{},
		Units:       map[string]*Unit{},
		Resources:   map[string]*economydom.Resources{},
		Buildings:   []*Building{},
		Timeline:    timeline.New(),
		Status:      "active",
		NextRecruit: map[string]*RecruitOrder{},
		NextBuild:   map[string]*BuildOrder{},
		Diplomacy:   diplomacydom.New(),
	}

	for _, p := range m.Provinces {
		match.Provinces[p.ID] = &Province{ID: p.ID, X: p.X, Y: p.Y}
		if p.HomeSlot != "" {
			match.Provinces[p.ID].Owner = p.HomeSlot
		}
	}

	for _, s := range m.Slots {
		userID := slotAssignments[s.ID]
		match.Players[s.ID] = &Player{
			Slot:   s.ID,
			UserID: userID,
			Color:  s.Color,
			Alive:  true,
		}
		bank := balance.StartingResources()
		match.Resources[s.ID] = &bank
		if prov, ok := match.Provinces[s.Capital]; ok {
			prov.Owner = s.ID
			prov.Capital = true
		}
	}

	match.Timeline.Push(&timeline.Event{
		At:   startedAt.Add(balance.MacroPulseInterval),
		Kind: timeline.MacroPulse,
	})

	for _, su := range m.StartingUnits {
		prov, err := m.Province(su.Province)
		if err != nil {
			return nil, fmt.Errorf("starting unit on unknown province %q", su.Province)
		}
		hp := su.HP
		if hp == 0 {
			hp = 100
		}
		u := &Unit{
			ID:        uuid.New().String(),
			OwnerSlot: su.Slot,
			Type:      su.Type,
			HP:        hp,
			Origin:    prov.ID,
			Dest:      prov.ID,
			OriginX:   prov.X,
			OriginY:   prov.Y,
			DestX:     prov.X,
			DestY:     prov.Y,
			Speed:     defaultSpeed(su.Type),
			StartedAt: startedAt,
			ArrivesAt: startedAt,
			Version:   1,
		}
		match.Units[u.ID] = u
	}
	return match, nil
}

// defaultSpeed gives a unit type a baseline pixel/game-second velocity,
// resolved through the [unitdom] catalogue.
func defaultSpeed(unitType string) float64 {
	if k, ok := unitdom.ByID(unitType); ok {
		return k.Speed()
	}
	if k, ok := unitdom.ByID("infantry"); ok {
		return k.Speed()
	}
	return balance.FallbackUnitSpeed
}
