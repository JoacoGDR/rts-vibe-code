package matchdom

import "github.com/joaquing/clone-supremacy/pkg/shared/wire"

// UnitToWire projects a unit into the wire snapshot shape.
func UnitToWire(u *Unit, x, y float64) wire.UnitState {
	out := wire.UnitState{
		ID: u.ID, OwnerID: u.OwnerSlot, Type: u.Type,
		X: x, Y: y, HP: u.HP,
		Origin: u.Origin, Dest: u.Dest,
		StartedAt: u.StartedAt, ArrivesAt: u.ArrivesAt,
		PathIndex: u.PathIndex,
	}
	if len(u.Path) > 0 {
		out.Path = make([]wire.PathLegState, len(u.Path))
		for i, l := range u.Path {
			out.Path[i] = wire.PathLegState{
				FromProv: l.FromProv, ToProv: l.ToProv,
				FromX: l.FromX, FromY: l.FromY, ToX: l.ToX, ToY: l.ToY,
				ArrivesAt: l.ArrivesAt,
			}
		}
	}
	return out
}
