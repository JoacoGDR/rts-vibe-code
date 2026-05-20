package visibility

import (
	"fmt"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/balance"
	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
	"github.com/joaquing/clone-supremacy/internal/domain/economydom"
	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/timeline"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// BuildSyntheticMatch constructs a grid map with the given province count,
// slots, and units for benchmarks and tests.
func BuildSyntheticMatch(provinceCount, slotCount, unitsPerSlot int) *matchdom.Match {
	side := 1
	for side*side < provinceCount {
		side++
	}
	edges := make([]maps.Edge, 0)
	provinces := make([]maps.Province, 0, provinceCount)
	provIDs := make([]string, 0, provinceCount)
	for i := 0; i < provinceCount; i++ {
		id := fmt.Sprintf("p%03d", i)
		provIDs = append(provIDs, id)
		col := i % side
		row := i / side
		provinces = append(provinces, maps.Province{
			ID: id,
			X:  float64(col) * 100,
			Y:  float64(row) * 100,
		})
	}
	for i := 0; i < provinceCount; i++ {
		if i%side != side-1 {
			edges = append(edges, maps.Edge{From: provIDs[i], To: provIDs[i+1]})
		}
		if i+side < provinceCount {
			edges = append(edges, maps.Edge{From: provIDs[i], To: provIDs[i+side]})
		}
	}
	slots := make([]maps.Slot, slotCount)
	slotIDs := make([]string, slotCount)
	for i := 0; i < slotCount; i++ {
		sid := fmt.Sprintf("s%03d", i)
		slotIDs[i] = sid
		slots[i] = maps.Slot{ID: sid, Color: sid, Capital: provIDs[i%provinceCount]}
	}
	mapDef := &maps.Map{
		ID:        "synthetic",
		Provinces: provinces,
		Edges:     edges,
		Slots:     slots,
	}
	now := time.Now()
	m := &matchdom.Match{
		ID:          "bench",
		MapID:       "synthetic",
		Map:         mapDef,
		StartedAt:   now,
		GameStart:   now,
		GameNow:     now,
		Speed:       60,
		Players:     map[string]*matchdom.Player{},
		Provinces:   map[string]*matchdom.Province{},
		Units:       map[string]*matchdom.Unit{},
		Resources:   map[string]*economydom.Resources{},
		Timeline:    timeline.New(),
		Status:      "active",
		NextRecruit: map[string]*matchdom.RecruitOrder{},
		NextBuild:   map[string]*matchdom.BuildOrder{},
		Diplomacy:   diplomacydom.New(),
	}
	for i, sid := range slotIDs {
		m.Players[sid] = &matchdom.Player{Slot: sid, Alive: true}
		bank := balance.StartingResources()
		m.Resources[sid] = &bank
		pid := provIDs[i%provinceCount]
		m.Provinces[pid] = &matchdom.Province{ID: pid, X: provinces[i%provinceCount].X, Y: provinces[i%provinceCount].Y, Owner: sid}
	}
	for _, p := range provinces {
		if _, ok := m.Provinces[p.ID]; !ok {
			m.Provinces[p.ID] = &matchdom.Province{ID: p.ID, X: p.X, Y: p.Y}
		}
	}
	uid := 0
	for si, sid := range slotIDs {
		for u := 0; u < unitsPerSlot; u++ {
			pid := provIDs[(si*unitsPerSlot+u)%provinceCount]
			prov := m.Provinces[pid]
			id := fmt.Sprintf("u%05d", uid)
			uid++
			m.Units[id] = &matchdom.Unit{
				ID: id, OwnerSlot: sid, Type: "infantry",
				HP: 10, Origin: pid, Dest: pid,
				OriginX: prov.X, OriginY: prov.Y, DestX: prov.X, DestY: prov.Y,
			}
		}
	}
	return m
}
