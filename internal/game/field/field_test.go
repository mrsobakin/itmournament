package field_test

import (
	"bytes"
	_ "embed"
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrsobakin/itmournament/internal/game/field"
)

//go:embed testdata/field.txt
var txtFuzzField []byte

//go:embed testdata/dense.txt
var txtDenseField []byte

func RunFieldTests(t *testing.T, newField func() field.Field) {
	t.Run("Load_ValidConfiguration", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     10,
			H:     10,
			Sizes: [4]int64{1, 1, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 0, Size: 2, IsVert: false},
			{X: 9, Y: 9, Size: 1, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.NoError(t, err, "expected no error for valid configuration")
	})

	t.Run("Load_MismatchedShipCounts", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     5,
			H:     5,
			Sizes: [4]int64{2, 0, 1, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 0, Size: 1, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for mismatched ship counts")
	})

	// . . .
	// . A A x x
	// . . .
	t.Run("Load_OutOfBoundsShips", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{0, 0, 0, 1},
		}

		ships := slices.Values([]field.Ship{
			{X: 1, Y: 1, Size: 4, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for out-of-bounds ships")
	})

	// . . .
	// . . . A
	// . . .
	t.Run("Load_OutOfBoundsShips", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{1, 0, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 3, Y: 1, Size: 4, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for out-of-bounds ships")
	})

	// . . B . .
	// A A X A .
	// . . B . .
	// . . B . .
	// . . . . .
	t.Run("Load_IntersectingShips", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     5,
			H:     5,
			Sizes: [4]int64{0, 0, 0, 2},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 1, Size: 4, IsVert: false},
			{X: 2, Y: 0, Size: 4, IsVert: true},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for intersecting ships")
	})

	// A . .
	// . B .
	// . . .
	t.Run("Load_ShipsTouchingCorners", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{2, 0, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 0, Size: 1, IsVert: false},
			{X: 1, Y: 1, Size: 1, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for ships touching borders")
	})

	// . . . . .
	// . . . . .
	// A A A . .
	// . . . B .
	// . . . B .
	t.Run("Load_MulticellShipsTouchingCorners", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     5,
			H:     5,
			Sizes: [4]int64{0, 1, 1, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 2, Size: 3, IsVert: false},
			{X: 3, Y: 3, Size: 2, IsVert: true},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for ships touching borders")
	})

	// . . . . .
	// . . . . .
	// B B B . .
	// . . . A .
	// . . . A .
	t.Run("Load_MulticellShipsTouchingCorners", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     5,
			H:     5,
			Sizes: [4]int64{0, 1, 1, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 3, Y: 3, Size: 2, IsVert: true},
			{X: 0, Y: 2, Size: 3, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for ships touching borders")
	})

	// . . .
	// . B A
	// . . .
	t.Run("Load_ShipsTouchingBorders", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{2, 0, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 2, Y: 1, Size: 1, IsVert: false},
			{X: 1, Y: 1, Size: 1, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for ships touching borders")
	})

	// . . . . .
	// A A A . .
	// . . B B .
	// . . . . .
	// . . . . .
	t.Run("Load_MulticellShipsTouchingBorders", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     5,
			H:     5,
			Sizes: [4]int64{0, 1, 1, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 1, Size: 3, IsVert: false},
			{X: 2, Y: 2, Size: 2, IsVert: false},
		})

		err := f.Load(conf, ships)
		assert.Error(t, err, "expected error for ships touching borders")
	})

	// . . .
	// . . .
	// . . .
	t.Run("Shoot_EmptyCell", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{W: 3, H: 3, Sizes: [4]int64{0, 0, 0, 0}}
		require.NoError(t, f.Load(conf, slices.Values([]field.Ship{})))

		result := f.Shoot(1, 1)
		assert.Equal(t, field.Miss, result, "expected Miss for empty cell")
	})

	// . . .
	// . A .
	// . . .
	t.Run("Shoot_DestroyShip", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{1, 0, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 1, Y: 1, Size: 1, IsVert: false},
		})

		require.NoError(t, f.Load(conf, ships))

		result := f.Shoot(1, 1)
		assert.Equal(t, field.Kill, result, "expected Kill for destroying a ship")
	})

	// . . .
	// . A .
	// . . .
	t.Run("Shoot_Repeatedly", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{1, 0, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 1, Y: 1, Size: 1, IsVert: false},
		})

		require.NoError(t, f.Load(conf, ships))

		result := f.Shoot(1, 1)
		assert.Equal(t, field.Kill, result, "expected Hit on first shot")

		result = f.Shoot(1, 1)
		assert.Equal(t, field.Kill, result, "expected Kill on repeated shot at same cell")
	})

	// A . .
	// A . .
	// . . .
	t.Run("Shoot_DestroyMulticellShip", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     3,
			H:     3,
			Sizes: [4]int64{0, 1, 0, 0},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 0, Size: 2, IsVert: true},
		})

		require.NoError(t, f.Load(conf, ships))

		assert.Equal(t, field.Hit, f.Shoot(0, 0), "expected Hit on first shot at multicell ship")
		assert.Equal(t, field.Kill, f.Shoot(0, 1), "expected Kill when entire multicell ship is destroyed")
	})

	// A A A A . B
	// . . . . . B
	// E . . F . B
	// E . . . . B
	// . . . . . .
	// D D D . C C
	t.Run("Shoot_RealField", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     6,
			H:     6,
			Sizes: [4]int64{1, 2, 1, 2},
		}

		ships := slices.Values([]field.Ship{
			{X: 0, Y: 0, Size: 4, IsVert: false},
			{X: 5, Y: 0, Size: 4, IsVert: true},
			{X: 4, Y: 5, Size: 2, IsVert: false},
			{X: 0, Y: 5, Size: 3, IsVert: false},
			{X: 0, Y: 2, Size: 2, IsVert: true},
			{X: 3, Y: 2, Size: 1, IsVert: false},
		})

		require.NoError(t, f.Load(conf, ships))

		assert.False(t, f.AllDead())

		// Some empty cells
		assert.Equal(t, field.Miss, f.Shoot(4, 0))
		assert.Equal(t, field.Miss, f.Shoot(0, 1))
		assert.Equal(t, field.Miss, f.Shoot(3, 1))
		assert.Equal(t, field.Miss, f.Shoot(3, 5))
		assert.False(t, f.AllDead())

		// E
		assert.Equal(t, field.Hit, f.Shoot(0, 2))
		assert.Equal(t, field.Kill, f.Shoot(0, 3))
		assert.False(t, f.AllDead())

		// B
		assert.Equal(t, field.Hit, f.Shoot(5, 2))
		assert.Equal(t, field.Hit, f.Shoot(5, 0))
		assert.Equal(t, field.Hit, f.Shoot(5, 3))
		assert.Equal(t, field.Kill, f.Shoot(5, 1))
		assert.False(t, f.AllDead())

		// F
		assert.Equal(t, field.Kill, f.Shoot(3, 2))
		assert.False(t, f.AllDead())

		// C
		assert.Equal(t, field.Hit, f.Shoot(4, 5))
		assert.Equal(t, field.Kill, f.Shoot(5, 5))
		assert.False(t, f.AllDead())

		// A
		assert.Equal(t, field.Hit, f.Shoot(3, 0))
		assert.Equal(t, field.Hit, f.Shoot(1, 0))
		assert.Equal(t, field.Hit, f.Shoot(2, 0))
		assert.Equal(t, field.Kill, f.Shoot(0, 0))
		assert.False(t, f.AllDead())

		// D
		assert.Equal(t, field.Hit, f.Shoot(2, 5))
		assert.False(t, f.AllDead())
		assert.Equal(t, field.Hit, f.Shoot(0, 5))
		assert.False(t, f.AllDead())
		assert.Equal(t, field.Kill, f.Shoot(1, 5))

		assert.True(t, f.AllDead())
	})

	t.Run("Shoot_Fuzzy", func(t *testing.T) {
		f := newField()
		conf := field.Configuration{
			W:     500,
			H:     500,
			Sizes: [4]int64{14800, 11100, 7400, 3700},
		}

		ships := field.ParseShips(bytes.NewReader(txtFuzzField))

		require.NoError(t, f.Load(conf, ships))

		type pos struct{ x, y int }
		var coordinates []pos

		for x := 0; x < 500; x++ {
			for y := 0; y < 500; y++ {
				coordinates = append(coordinates, pos{x, y})
			}
		}

		rand.Shuffle(len(coordinates), func(i, j int) {
			coordinates[i], coordinates[j] = coordinates[j], coordinates[i]
		})

		var nHits, nKills int

		for _, coord := range coordinates {
			res := f.Shoot(int64(coord.x), int64(coord.y))

			switch res {
			case field.Hit:
				nHits++
			case field.Kill:
				nKills++
			}
		}

		nExpectedKills := int(conf.Sizes[0] + conf.Sizes[1] + conf.Sizes[2] + conf.Sizes[3])
		assert.Equal(t, nExpectedKills, nKills)

		nExpectedHits := int(conf.Sizes[1] + 2*conf.Sizes[2] + 3*conf.Sizes[3])
		assert.Equal(t, nExpectedHits, nHits)
	})

	// Dense field is a such field, that moving any ship
	// anywhere but its original position makes it invalid.
	t.Run("Load_Dense", func(t *testing.T) {
		conf := field.Configuration{
			W:     10,
			H:     10,
			Sizes: [4]int64{2, 8, 3, 4},
		}
		shipsOriginal := slices.Collect(field.ParseShips(bytes.NewReader(txtDenseField)))

		{
			f := newField()
			require.NoError(t, f.Load(conf, slices.Values(shipsOriginal)), "original field should load")
		}

		for i, ship := range shipsOriginal {
			var displaced []field.Ship
			displaced = append(displaced, shipsOriginal...)

			var maxX, maxY int64
			if ship.IsVert {
				maxX = 10
				maxY = 10 - int64(ship.Size) + 1
			} else {
				maxX = 10 - int64(ship.Size) + 1
				maxY = 10
			}

			for x := int64(0); x < maxX; x++ {
				for y := int64(0); y < maxY; y++ {
					if ship.X == int64(x) && ship.Y == int64(y) {
						continue
					}

					displaced[i].X = x
					displaced[i].Y = y

					f := newField()
					assert.Error(t, f.Load(conf, slices.Values(displaced)), "displaced field should not load")
				}
			}
		}
	})
}

func TestShipField(t *testing.T) {
	RunFieldTests(t, func() field.Field {
		return field.NewShipField(0)
	})
}
