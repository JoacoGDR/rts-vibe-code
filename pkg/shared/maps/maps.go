// Package maps owns the static map/scenario definitions consumed by the engine.
// Maps are authored as YAML documents and embedded at build time. Each map
// defines a graph of provinces, an edge list (with distances), starting
// player slots and starting unit deployments.
package maps

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var fsys embed.FS

type Map struct {
	ID            string         `yaml:"id" json:"id"`
	Name          string         `yaml:"name" json:"name"`
	Provinces     []Province     `yaml:"provinces" json:"provinces"`
	Edges         []Edge         `yaml:"edges" json:"edges"`
	Slots         []Slot         `yaml:"slots" json:"slots"`
	StartingUnits []StartingUnit `yaml:"starting_units" json:"starting_units"`
}

type Province struct {
	ID   string  `yaml:"id" json:"id"`
	X    float64 `yaml:"x" json:"x"`
	Y    float64 `yaml:"y" json:"y"`
	Name string  `yaml:"name,omitempty" json:"name,omitempty"`
}

type Edge struct {
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to" json:"to"`
}

type Slot struct {
	ID      string `yaml:"id" json:"id"`
	Color   string `yaml:"color" json:"color"`
	Capital string `yaml:"capital" json:"capital"`
}

type StartingUnit struct {
	Slot     string  `yaml:"slot" json:"slot"`
	Type     string  `yaml:"type" json:"type"`
	Province string  `yaml:"province" json:"province"`
	HP       float64 `yaml:"hp" json:"hp,omitempty"`
}

// Distance returns euclidean distance between two provinces.
func (m *Map) Distance(a, b string) (float64, error) {
	pa, err := m.Province(a)
	if err != nil {
		return 0, err
	}
	pb, err := m.Province(b)
	if err != nil {
		return 0, err
	}
	dx := pa.X - pb.X
	dy := pa.Y - pb.Y
	return sqrt(dx*dx + dy*dy), nil
}

func (m *Map) Province(id string) (Province, error) {
	for _, p := range m.Provinces {
		if p.ID == id {
			return p, nil
		}
	}
	return Province{}, fmt.Errorf("province %q not on map", id)
}

func (m *Map) HasEdge(a, b string) bool {
	for _, e := range m.Edges {
		if (e.From == a && e.To == b) || (e.From == b && e.To == a) {
			return true
		}
	}
	return false
}

func (m *Map) Neighbors(id string) []string {
	seen := map[string]struct{}{}
	for _, e := range m.Edges {
		switch {
		case e.From == id:
			seen[e.To] = struct{}{}
		case e.To == id:
			seen[e.From] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Load parses a map by ID from the embedded set.
func Load(id string) (*Map, error) {
	files, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		raw, err := fsys.ReadFile(f.Name())
		if err != nil {
			continue
		}
		var m Map
		if err := yaml.Unmarshal(raw, &m); err != nil {
			continue
		}
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("map %q not found", id)
}

// All lists every embedded map's metadata.
func All() ([]Map, error) {
	files, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	out := make([]Map, 0, len(files))
	for _, f := range files {
		raw, err := fsys.ReadFile(f.Name())
		if err != nil {
			continue
		}
		var m Map
		if err := yaml.Unmarshal(raw, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// sqrt is a tiny helper to avoid pulling in math just for one call.
func sqrt(v float64) float64 {
	if v <= 0 {
		return 0
	}
	z := v
	for i := 0; i < 20; i++ {
		z = (z + v/z) / 2
	}
	return z
}
