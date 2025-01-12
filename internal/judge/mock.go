package judge

import (
	"fmt"
	"strings"

	"github.com/mrsobakin/itmournament/internal/game/field"
)

type mockPlayer struct {
	conf  field.Configuration
	field field.Field

	x, y int64
}

func newMockMaster(field field.Field, conf field.Configuration) *mockPlayer {
	return &mockPlayer{
		conf:  conf,
		field: field,
		x:     0,
		y:     0,
	}

}

func (p *mockPlayer) SendCommand(cmd string) (string, error) {
	switch cmd {
	case "get count 1":
		return fmt.Sprint(p.conf.Sizes[0]), nil
	case "get count 2":
		return fmt.Sprint(p.conf.Sizes[1]), nil
	case "get count 3":
		return fmt.Sprint(p.conf.Sizes[2]), nil
	case "get count 4":
		return fmt.Sprint(p.conf.Sizes[3]), nil
	case "get width":
		return fmt.Sprint(p.conf.W), nil
	case "get height":
		return fmt.Sprint(p.conf.H), nil
	case "win":
		return "no", nil
	case "shot":
		return fmt.Sprintf("%d %d", p.x, p.y), nil
	}

	if strings.HasPrefix(cmd, "shot ") {
		var x, y int64
		fmt.Sscanf(cmd, "shot %d %d", &x, &y)

		result := p.field.Shoot(x, y)
		return result.String(), nil
	}

	switch cmd {
	case "set result miss":
		// Good. We found an empty cell.
	case "set result hit", "Set result kill":
		p.x += 1

		if p.x == p.conf.W {
			p.x = 0
			p.y += 1
		}

		if p.y == p.conf.H {
			p.y = 0
		}
	}

	return "ok", nil
}

func (p *mockPlayer) RetrieveField(field.Configuration) (field.Field, error) {
	return p.field, nil
}

func (p *mockPlayer) Close() error {
	return nil
}
