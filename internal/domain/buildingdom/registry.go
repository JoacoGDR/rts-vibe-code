package buildingdom

var registry = map[string]Kind{
	"factory": factory{},
}

// ByID looks up a building kind by its string id.
func ByID(id string) (Kind, bool) {
	k, ok := registry[id]
	return k, ok
}

// All returns every registered kind.
func All() []Kind {
	out := make([]Kind, 0, len(registry))
	for _, k := range registry {
		out = append(out, k)
	}
	return out
}
