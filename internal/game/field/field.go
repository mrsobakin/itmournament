package field

import (
	"bufio"
	"fmt"
	"io"
	"iter"
)

type ShootResult int

const (
	Miss ShootResult = iota
	Hit
	Kill
)

func (r *ShootResult) FromString(str string) error {
	switch str {
	case "miss":
		*r = Miss
	case "hit":
		*r = Hit
	case "kill":
		*r = Kill
	default:
		return fmt.Errorf("invalid shoot result")
	}
	return nil
}

func (r ShootResult) String() string {
	switch r {
	case Miss:
		return "miss"
	case Hit:
		return "hit"
	case Kill:
		return "kill"
	default:
		panic("invalid shoot result")
	}
}

type Configuration struct {
	W, H  int64
	Sizes [4]int64
}

func (c *Configuration) IsValid() error {
	if c.W <= 0 || c.H <= 0 {
		return fmt.Errorf("non-positive field size: [%d %d]", c.W, c.H)
	}

	if c.Sizes[0] < 0 || c.Sizes[1] < 0 || c.Sizes[2] < 0 || c.Sizes[3] < 0 {
		return fmt.Errorf("negative ship amount: [%d %d %d %d]", c.Sizes[0], c.Sizes[1], c.Sizes[2], c.Sizes[3])
	}

	if (c.Sizes[0] + c.Sizes[1] + c.Sizes[2] + c.Sizes[3]) <= 0 {
		return fmt.Errorf("summary ship count is non-positive: [%d %d %d %d]", c.Sizes[0], c.Sizes[1], c.Sizes[2], c.Sizes[3])
	}

	return nil
}

type Ship struct {
	X, Y   int64
	Size   int8
	IsVert bool
}

type Field interface {
	// Loads field given ship sequence and configuration.
	//
	// If field is invalid, i.e. has ships intersecting,
	// exceeds field size or ship conunt does not match
	// given configuration, returns an error.
	Load(Configuration, iter.Seq[Ship]) error

	// Emulates a shot, modifies field internal state and
	// returns the expected result of a shot.
	Shoot(x, y int64) ShootResult

	// Undoes all shots on the field, i.e. reverts field
	// to the state just after `Load`.
	ResetShots()

	// Returns whether all ships are destroyed, i.e. the
	// corresponding player lost.
	AllDead() bool
}

func ParseShips(src io.Reader) iter.Seq[Ship] {
	return func(yield func(s Ship) bool) {
		lines := bufio.NewScanner(src)

		// Skip first line with field dimensions
		lines.Scan()

		for lines.Scan() {
			var ship Ship
			var direction rune

			n, err := fmt.Sscanf(lines.Text(), "%d %c %d %d", &ship.Size, &direction, &ship.X, &ship.Y)

			if err != nil || n != 4 {
				return
			}

			switch direction {
			case 'v':
				ship.IsVert = true
			case 'h':
				ship.IsVert = false
			default:
				return
			}

			if !yield(ship) {
				return
			}
		}
	}
}
