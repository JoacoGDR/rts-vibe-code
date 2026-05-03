package unitdom

// registry is the canonical catalogue of every unit type the simulation
// knows about. Adding a new unit kind = one new file + one line below.
var registry = map[string]Kind{
	"infantry": infantry{},
	"cavalry":  cavalry{},
	"armor":    armor{},
}

// ByID looks up a unit kind by its string id. The boolean reports whether
// the kind was registered.
func ByID(id string) (Kind, bool) {
	k, ok := registry[id]
	return k, ok
}

// All returns every registered kind. Order is not stable.
func All() []Kind {
	out := make([]Kind, 0, len(registry))
	for _, k := range registry {
		out = append(out, k)
	}
	return out
}
