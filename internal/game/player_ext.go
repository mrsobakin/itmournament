package game

import (
	"fmt"

	"github.com/mrsobakin/itmournament/internal/game/field"
)

type PlayerExt struct {
	Player
}

func (p *PlayerExt) Sendf(format string, a ...any) (string, error) {
	return p.SendCommand(fmt.Sprintf(format, a...))
}

func (p *PlayerExt) SendScanf(cmd string, format string, a ...any) error {
	resp, err := p.SendCommand(cmd)
	if err != nil {
		return err
	}

	n, err := fmt.Sscanf(resp, format, a...)
	if err != nil {
		return err
	}
	if n != len(a) {
		return fmt.Errorf("response does not match format")
	}

	return nil
}

func (p *PlayerExt) RequestAndGetField(conf field.Configuration) (field.Field, error) {
	resp, err := p.SendCommand("dump /tmp/field.txt")
	if err != nil {
		return nil, err
	}
	if resp != "ok" {
		return nil, fmt.Errorf("did not dump field")
	}

	return p.RetrieveField(conf)
}
