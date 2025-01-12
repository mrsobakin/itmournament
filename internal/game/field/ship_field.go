package field

import (
	"errors"
	"iter"

	"github.com/dolthub/swiss"
)

type shipData uint8 // 1:1:2:4 <-> 1:IsVert:Size:Cells

func newShipData(isVert bool, size int8) shipData {
	var bits = (uint8(size) - 1) << 4

	bits |= (0b1111 << size) & 0b1111

	if isVert {
		return shipData(0b11000000 | bits)
	} else {
		return shipData(0b10000000 | bits)
	}
}

func (c shipData) IsVert() bool {
	return ((c >> 6) & 0b1) != 0
}

func (c shipData) IsHor() bool {
	return !c.IsVert()
}

func (c shipData) Size() int8 {
	return int8((c>>4)&0b11) + 1
}

func (c *shipData) MarkHit(idx int8) {
	*c |= (1 << idx)
}

func (c shipData) IsDead() bool {
	return (c & 0b1111) == 0b1111
}

type packedPos int64

type intersection struct {
	shipPos  packedPos
	shipData shipData
	deck     int8
}

type ShipField struct {
	ships      *swiss.Map[packedPos, shipData]
	conf       Configuration
	currConfig Configuration
}

func NewShipField(sizeHint uint32) *ShipField {
	return &ShipField{
		ships: swiss.NewMap[packedPos, shipData](sizeHint),
	}
}

func (f *ShipField) makePos(x, y int64) packedPos {
	return packedPos(x*f.conf.H + y)
}

func (f *ShipField) hitShip(ship shipData, pos packedPos, idx int8) ShootResult {
	if ship.IsDead() {
		return Kill
	}

	ship.MarkHit(idx)

	f.ships.Put(pos, ship)

	if ship.IsDead() {
		f.currConfig.Sizes[ship.Size()-1] -= 1
		return Kill
	} else {
		return Hit
	}
}

func (f *ShipField) scanIntersectionsLeft(x, y int64) (intersection, bool) {
	for i := int8(1); i <= int8(min(3, x)); i++ {
		pos := f.makePos(x-int64(i), y)
		if ship, exists := f.ships.Get(pos); exists && ship.IsHor() {
			if ship.Size() <= i {
				continue
			}

			return intersection{
				shipPos:  pos,
				shipData: ship,
				deck:     i,
			}, true
		}
	}

	var intersection intersection
	return intersection, false
}

func (f *ShipField) scanIntersectionsUp(x, y int64) (intersection, bool) {
	for i := int8(1); i <= int8(min(3, y)); i++ {
		pos := f.makePos(x, y-int64(i))
		if ship, exists := f.ships.Get(pos); exists && ship.IsVert() {
			if ship.Size() <= i {
				continue
			}

			return intersection{
				shipPos:  pos,
				shipData: ship,
				deck:     i,
			}, true
		}
	}

	var intersection intersection
	return intersection, false
}

// Checks whether there are ships that take start inside the given ship or its border.
// Returns true if there are such ships, i.e. there are intersections.
func (f *ShipField) checkInnerOverlaps(ship Ship) bool {
	minX := max(0, ship.X-1)
	minY := max(0, ship.Y-1)

	var maxX, maxY int64
	if ship.IsVert {
		maxX = min(f.conf.W-1, ship.X+1)
		maxY = min(f.conf.H-1, ship.Y+int64(ship.Size))
	} else {
		maxX = min(f.conf.W-1, ship.X+int64(ship.Size))
		maxY = min(f.conf.H-1, ship.Y+1)
	}

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			if f.ships.Has(f.makePos(x, y)) {
				return true
			}
		}
	}

	return false
}

func (f *ShipField) checkOuterLeftOverlaps(ship Ship) bool {
	x := max(0, ship.X-1)
	minY := max(0, ship.Y-1)

	var maxY int64
	if ship.IsVert {
		maxY = min(f.conf.H-1, ship.Y+int64(ship.Size))
	} else {
		maxY = min(f.conf.H-1, ship.Y+1)
	}

	for y := minY; y <= maxY; y++ {
		if _, found := f.scanIntersectionsLeft(x, y); found {
			return true
		}
	}

	return false
}

func (f *ShipField) checkOuterUpOverlaps(ship Ship) bool {
	y := max(0, ship.Y-1)
	minX := max(0, ship.X-1)

	var maxX int64
	if ship.IsVert {
		maxX = min(f.conf.W-1, ship.X+1)
	} else {
		maxX = min(f.conf.W-1, ship.X+int64(ship.Size))
	}

	for x := minX; x <= maxX; x++ {
		if _, found := f.scanIntersectionsUp(x, y); found {
			return true
		}
	}

	return false
}

func (f *ShipField) Load(conf Configuration, ships iter.Seq[Ship]) error {
	cellCounts := make([]int64, len(conf.Sizes))
	f.currConfig = conf
	f.conf = conf

	if conf.W == 0 || conf.H == 0 {
		return errors.New("invalid field size")
	}

	for ship := range ships {
		if ship.Size <= 0 || int(ship.Size) > len(conf.Sizes) {
			return errors.New("invalid ship size")
		}

		if ship.IsVert {
			if ship.X >= conf.W || ship.Y+int64(ship.Size) > conf.H {
				return errors.New("ship out of bounds")
			}
		} else {
			if ship.Y >= conf.H || ship.X+int64(ship.Size) > conf.W {
				return errors.New("ship out of bounds")
			}
		}

		if f.checkInnerOverlaps(ship) || f.checkOuterLeftOverlaps(ship) || f.checkOuterUpOverlaps(ship) {
			return errors.New("ships overlap")
		}

		pos := f.makePos(ship.X, ship.Y)
		f.ships.Put(pos, newShipData(ship.IsVert, ship.Size))

		cellCounts[ship.Size-1]++
	}

	for i, count := range cellCounts {
		if count != conf.Sizes[i] {
			return errors.New("ship count does not match configuration")
		}
	}

	return nil
}

func (f *ShipField) Shoot(x, y int64) ShootResult {
	if x >= f.conf.W || y >= f.conf.H {
		return Miss
	}

	pos := f.makePos(x, y)
	if ship, exists := f.ships.Get(pos); exists {
		return f.hitShip(ship, pos, 0)
	}

	if intr, found := f.scanIntersectionsLeft(x, y); found {
		return f.hitShip(intr.shipData, intr.shipPos, intr.deck)
	}

	if intr, found := f.scanIntersectionsUp(x, y); found {
		return f.hitShip(intr.shipData, intr.shipPos, intr.deck)
	}

	return Miss
}

func (f *ShipField) ResetShots() {
	f.currConfig = f.conf
	f.ships.Iter(func(pos packedPos, oldShip shipData) (stop bool) {
		f.ships.Put(pos, newShipData(oldShip.IsVert(), oldShip.Size()))
		return
	})
}

func (f *ShipField) AllDead() bool {
	return f.currConfig.Sizes[0] == 0 &&
		f.currConfig.Sizes[1] == 0 &&
		f.currConfig.Sizes[2] == 0 &&
		f.currConfig.Sizes[3] == 0
}
