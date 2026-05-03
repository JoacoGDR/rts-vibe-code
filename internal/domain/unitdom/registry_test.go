package unitdom

import "testing"

func TestRPSDamageTable(t *testing.T) {
	cases := []struct {
		atk, def string
		want     float64
	}{
		{"infantry", "armor", 38},
		{"armor", "cavalry", 38},
		{"cavalry", "infantry", 38},
		{"infantry", "cavalry", 22},
		{"infantry", "infantry", 30},
	}
	for _, c := range cases {
		atk, _ := ByID(c.atk)
		def, _ := ByID(c.def)
		got := atk.DamageVs(def)
		if got != c.want {
			t.Errorf("%s vs %s: got %f, want %f", c.atk, c.def, got, c.want)
		}
	}
}

func TestArmorRequiresFactoryBuilding(t *testing.T) {
	k, ok := ByID("armor")
	if !ok {
		t.Fatal("armor missing from registry")
	}
	if k.NeedsBuilding() != "factory" {
		t.Fatalf("armor needs factory, got %q", k.NeedsBuilding())
	}
}
