package game

import (
	"context"
	"time"

	"github.com/mrsobakin/itmournament/internal/game/field"
	"github.com/mrsobakin/itmournament/internal/utils"
)

type Role int

const (
	RoleMaster Role = iota
	RoleSlave
)

func (r Role) Other() Role {
	if r == RoleMaster {
		return RoleSlave
	} else {
		return RoleMaster
	}
}

func (r Role) String() string {
	if r == RoleMaster {
		return "master"
	} else {
		return "slave"
	}
}

type TerminationReason int

const (
	ReasonNormal TerminationReason = iota
	ReasonRuntimeError
	ReasonMemoryLimit
	ReasonTimeLimit
)

type ErrorTerminated struct {
	Reason TerminationReason
}

var (
	ErrTerminatedMemoryLimit error = &ErrorTerminated{ReasonMemoryLimit}
)

func (e *ErrorTerminated) Is(target error) bool {
	if err, ok := target.(*ErrorTerminated); ok {
		return e.Reason == err.Reason
	}

	return false
}

func (e *ErrorTerminated) Error() string {
	switch e.Reason {
	case ReasonNormal:
		return "player terminated normally"
	case ReasonRuntimeError:
		return "player terminated due to runtime error"
	case ReasonMemoryLimit:
		return "player terminated due to memory limit"
	case ReasonTimeLimit:
		return "player terminated due to time limit"
	default:
		panic("unknown termination reason")
	}
}

type Player interface {
	// Sends a command and receives a response for it.
	//
	// If Player was terminated, special error `ErrorTerminated`
	// will be returned.
	SendCommand(string) (string, error)

	// Retrieves player's field.
	//
	// If field doesn't match configuration or is invalid, error is retured.
	// Should be called ONLY after the corresponding command is executed.
	// MUST NOT be called more than once.
	//
	// If Player was terminated, special error `ErrorTerminated`
	// will be returned.
	RetrieveField(field.Configuration) (field.Field, error)

	// Terminates player session.
	Close() error
}

type PlayerFactory interface {
	NewPlayer(context.Context) Player
}

type StopwatchPlayer struct {
	player    Player
	stopwatch *utils.Stopwatch
}

func NewStopwatchPlayer(player Player, stopwatch *utils.Stopwatch) *StopwatchPlayer {
	return &StopwatchPlayer{
		player,
		stopwatch,
	}
}

type StopwatchPlayerFactory struct {
	playerFactory PlayerFactory
	timeout       time.Duration
	cause         error
}

func NewStopwatchPlayerFactory(playerFactory PlayerFactory, timeout time.Duration, cause error) PlayerFactory {
	return &StopwatchPlayerFactory{
		playerFactory,
		timeout,
		cause,
	}
}

func (f *StopwatchPlayerFactory) NewPlayer(ctx context.Context) Player {
	swCtx, sw := utils.NewStopwatchContext(ctx, f.timeout, f.cause)

	player := f.playerFactory.NewPlayer(swCtx)
	swPlayer := NewStopwatchPlayer(player, sw)

	return swPlayer
}

func (p *StopwatchPlayer) SendCommand(command string) (string, error) {
	p.stopwatch.Resume()
	defer p.stopwatch.Pause()
	return p.player.SendCommand(command)
}

func (p *StopwatchPlayer) RetrieveField(conf field.Configuration) (field.Field, error) {
	return p.player.RetrieveField(conf)
}

func (p *StopwatchPlayer) Close() error {
	return p.player.Close()
}
