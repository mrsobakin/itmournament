package judge

import (
	"errors"
	"fmt"

	"github.com/mrsobakin/itmournament/internal/game"
	"github.com/mrsobakin/itmournament/internal/game/field"
)

var (
	errPlayerWon error = errors.New("player won")
)

type roleError struct {
	Role game.Role
	Err  error
}

func failedAs(role game.Role, err error) *roleError {
	return &roleError{
		role,
		err,
	}
}

func wonAs(role game.Role) *roleError {
	return &roleError{
		role,
		errPlayerWon,
	}
}

type round struct {
	master, slave           game.PlayerExt
	masterField, slaveField field.Field
	conf                    field.Configuration
}

func newRound(master, slave game.Player) *round {
	return &round{
		master: game.PlayerExt{
			Player: master,
		},
		slave: game.PlayerExt{
			Player: slave,
		},
	}
}

func (r *round) playerByRole(role game.Role) game.PlayerExt {
	if role == game.RoleMaster {
		return r.master
	} else {
		return r.slave
	}
}

func (r *round) fieldByRole(role game.Role) field.Field {
	if role == game.RoleMaster {
		return r.masterField
	} else {
		return r.slaveField
	}
}

func (r *round) InitPlayer(role game.Role) *roleError {
	player := r.playerByRole(role)

	resp, err := player.SendCommand("create " + role.String())
	if err != nil {
		return failedAs(role, fmt.Errorf("failed to create role: %w", err))
	}
	if resp != "ok" {
		return failedAs(role, fmt.Errorf("failed to create role, returned: %q", resp))
	}

	resp, err = player.SendCommand("set strategy custom")
	if err != nil {
		return failedAs(role, fmt.Errorf("failed to set strategy: %w", err))
	}
	if resp != "ok" {
		return failedAs(role, fmt.Errorf("failed to set strategy, returned: %q", resp))
	}

	return nil
}

func (r *round) confParams() []struct {
	name string
	val  *int64
} {
	return []struct {
		name string
		val  *int64
	}{
		{"width", &r.conf.W},
		{"height", &r.conf.H},
		{"count 1", &r.conf.Sizes[0]},
		{"count 2", &r.conf.Sizes[1]},
		{"count 3", &r.conf.Sizes[2]},
		{"count 4", &r.conf.Sizes[3]},
	}
}

func (r *round) RequestConfiguration() *roleError {
	for _, arg := range r.confParams() {
		if err := r.master.SendScanf("get "+arg.name, "%d", arg.val); err != nil {
			return failedAs(game.RoleMaster, fmt.Errorf("failed to get configuration: %w", err))
		}
	}

	if err := r.conf.IsValid(); err != nil {
		return failedAs(game.RoleMaster, fmt.Errorf("invalid configuration: %w", err))
	}

	return nil
}

func (r *round) TransferConfiguration() *roleError {
	for _, arg := range r.confParams() {
		resp, err := r.slave.Sendf("set %s %d", arg.name, *arg.val)
		if err != nil {
			return failedAs(game.RoleSlave, fmt.Errorf("failed to set configuration: %w", err))
		}
		if resp != "ok" {
			return failedAs(game.RoleSlave, fmt.Errorf("failed to set configuration, returned: %q", resp))
		}
	}

	return nil
}

func (r *round) StartPlayer(role game.Role) *roleError {
	player := r.playerByRole(role)

	resp, err := player.SendCommand("start")
	if err != nil {
		return failedAs(role, fmt.Errorf("failed to start: %w", err))
	}
	if resp != "ok" {
		return failedAs(role, fmt.Errorf("failed to start, returned: %q", resp))
	}

	if role == game.RoleMaster {
		r.masterField, err = player.RequestAndGetField(r.conf)
	} else {
		r.slaveField, err = player.RequestAndGetField(r.conf)
	}
	if err != nil {
		return failedAs(role, fmt.Errorf("failed to get field: %w", err))
	}

	return nil
}

func (r *round) isValidShot(x, y int64) bool {
	return x >= 0 && y >= 0 && x < r.conf.W && y < r.conf.H
}

func (r *round) Shoot(shooterRole game.Role) (bool, *roleError) {
	victimRole := shooterRole.Other()

	shooter := r.playerByRole(shooterRole)
	victim := r.playerByRole(victimRole)

	victimField := r.fieldByRole(victimRole)

	var x, y int64
	if err := shooter.SendScanf("shot", "%d %d", &x, &y); err != nil {
		return false, failedAs(shooterRole, fmt.Errorf("failed to request shoot coordinaes: %w", err))
	}

	if !r.isValidShot(x, y) {
		return false, failedAs(shooterRole, fmt.Errorf("invalid shoot position %d %d", x, y))
	}

	resp, err := victim.Sendf("shot %d %d", x, y)
	if err != nil {
		return false, failedAs(victimRole, fmt.Errorf("failed to shoot: %w", err))
	}

	var result field.ShootResult
	if err := result.FromString(resp); err != nil {
		return false, failedAs(victimRole, err)
	}

	expectedResult := victimField.Shoot(x, y)

	if result != expectedResult {
		return false, failedAs(victimRole, fmt.Errorf("victim returned invalid shoot result: %d", result))
	}

	resp, err = shooter.Sendf("set result %s", resp)
	if err != nil {
		return false, failedAs(shooterRole, fmt.Errorf("failed to set shoot result: %w", err))
	}
	if resp != "ok" {
		return false, failedAs(shooterRole, fmt.Errorf("failed to set shoot result, returned: %q", resp))
	}

	var hit bool
	if !victimField.AllDead() {
		hit = (result != field.Miss)
		return hit, nil
	}

	return true, wonAs(shooterRole)
}

func (r *round) Judge() *roleError {
	if err := r.InitPlayer(game.RoleMaster); err != nil {
		return err
	}

	if err := r.InitPlayer(game.RoleSlave); err != nil {
		return err
	}

	if err := r.RequestConfiguration(); err != nil {
		return err
	}

	if err := r.StartPlayer(game.RoleMaster); err != nil {
		return err
	}

	// Just in case, configuration is transfered onto slave only after
	// master field is dumped and checked.
	// This prevents master from asking slave for fields it
	// itself is unable to generate.
	if err := r.TransferConfiguration(); err != nil {
		return err
	}

	if err := r.StartPlayer(game.RoleSlave); err != nil {
		return err
	}

	currentPlayer := game.RoleSlave
	for {
		hit, err := r.Shoot(currentPlayer)
		if err != nil {
			return err
		}

		if !hit {
			currentPlayer = currentPlayer.Other()
		}
	}
}
