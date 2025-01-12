package judge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mrsobakin/itmournament/internal/game"
	"github.com/mrsobakin/itmournament/internal/game/field"
)

var (
	errTimeoutGlobal = errors.New("global timeout")
	errTimeoutMaster = errors.New("master timeout")
	errTimeoutSlave  = errors.New("slave timeout")
)

type Result int

const (
	Tie Result = iota
	MasterWon
	SlaveWon
)

func (r Result) String() string {
	switch r {
	case Tie:
		return "tie"
	case MasterWon:
		return "master"
	case SlaveWon:
		return "slave"
	default:
		panic("invalid verdict")
	}
}

func (r Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func ResultFromWinner(role game.Role) Result {
	if role == game.RoleMaster {
		return MasterWon
	}
	if role == game.RoleSlave {
		return SlaveWon
	}
	panic("unknown role")
}

type Reason int

const (
	Ok Reason = iota
	RuntimeError
	MemoryLimit
	Timeout
	GlobalTimeout
)

func (r Reason) String() string {
	switch r {
	case Ok:
		return "OK"
	case RuntimeError:
		return "RE"
	case MemoryLimit:
		return "ML"
	case Timeout:
		return "TL"
	case GlobalTimeout:
		return "GTL"
	default:
		panic("invalid reason")
	}
}

func (r Reason) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

type Verdict struct {
	Winner  Result `json:"winner"`
	Reason  Reason `json:"reason"`
	Details string `json:"details"`
}

type Judge struct {
	PlayerTimeout time.Duration
	GlobalTimeout time.Duration
}

// As per our rules:
//   - If player wins, he wins.
//   - If player errors out, the other player wins.
//   - If slave loses due to ML, breaker round is
//     conducted to determine whether the master
//     is able to handle its own configuration.
//   - If he can't, it's a tie.
func (j *Judge) judgeMatch(ctx context.Context, masterFactory, slaveFactory game.PlayerFactory) (Result, error) {
	var masterField field.Field
	var conf field.Configuration

	{
		master := masterFactory.NewPlayer(ctx)
		slave := slaveFactory.NewPlayer(ctx)

		round := newRound(master, slave)

		result := round.Judge()
		if errors.Is(result.Err, errPlayerWon) {
			return ResultFromWinner(result.Role), nil
		}

		if !errors.Is(result.Err, game.ErrTerminatedMemoryLimit) {
			return ResultFromWinner(result.Role.Other()), result.Err
		}

		if result.Role == game.RoleMaster {
			return SlaveWon, result.Err
		}

		masterField = round.masterField
		masterField.ResetShots()
		conf = round.conf
	}

	mockMaster := newMockMaster(masterField, conf)

	master := masterFactory.NewPlayer(ctx)
	round := newRound(mockMaster, master)
	result := round.Judge()

	// If no errors on master side or master lost.
	if result.Role != game.RoleSlave {
		return ResultFromWinner(game.RoleMaster), nil
	}

	// If master won.
	if errors.Is(result.Err, errPlayerWon) {
		return ResultFromWinner(game.RoleMaster), nil
	}

	// If master had ANY error.
	err := fmt.Errorf("error during breaker round: %w", result.Err)
	return Tie, err
}

func (j *Judge) Judge(ctx context.Context, master, slave game.PlayerFactory) Verdict {
	swMaster := game.NewStopwatchPlayerFactory(master, j.PlayerTimeout, errTimeoutMaster)
	swSlave := game.NewStopwatchPlayerFactory(slave, j.PlayerTimeout, errTimeoutSlave)

	limitedCtx, cancel := context.WithTimeoutCause(ctx, j.GlobalTimeout, errTimeoutGlobal)
	defer cancel()

	verdict, details := j.judgeMatch(limitedCtx, swMaster, swSlave)

	reason := func() Reason {
		if errors.Is(details, errTimeoutGlobal) {
			return GlobalTimeout
		}

		if errors.Is(details, errTimeoutMaster) {
			verdict = SlaveWon
			return Timeout
		}

		if errors.Is(details, errTimeoutSlave) {
			verdict = MasterWon
			return Timeout
		}

		if errors.Is(details, game.ErrTerminatedMemoryLimit) {
			return MemoryLimit
		}

		if details != nil {
			return RuntimeError
		}

		return Ok
	}()

	detailsStr := ""
	if details != nil {
		detailsStr = details.Error()
	}

	return Verdict{
		Winner:  verdict,
		Reason:  reason,
		Details: detailsStr,
	}
}
