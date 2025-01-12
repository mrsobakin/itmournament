package docker

import (
	"bufio"
	"context"
	"errors"

	"github.com/mrsobakin/itmournament/internal/game"
	"github.com/mrsobakin/itmournament/internal/game/field"
)

func convertContainerErrToPlayerErr(err error) error {
	var contErr = &ErrorTerminated{}
	if !errors.As(err, &contErr) {
		return err
	}

	result := contErr.Result

	if result.Err != nil {
		return result.Err
	}

	var reason game.TerminationReason
	switch result.ExitCode {
	case 0:
		reason = game.ReasonNormal
	case 137:
		reason = game.ReasonMemoryLimit
	default:
		reason = game.ReasonRuntimeError
	}

	return &game.ErrorTerminated{Reason: reason}
}

type DockerPlayer struct {
	cont    *SubmissionContainer
	scanner *bufio.Scanner
}

func NewDockerPlayer(runner *SubmissionRunner, ctx context.Context, imageId string) (*DockerPlayer, error) {
	cont, err := runner.CreateSubmissionContainer(ctx, imageId)
	if err != nil {
		return nil, err
	}
	cont.Start()

	scanner := bufio.NewScanner(cont.Stdout)
	scanner.Split(bufio.ScanLines)

	p := &DockerPlayer{
		cont,
		scanner,
	}

	return p, nil
}

func (p *DockerPlayer) SendCommand(cmd string) (string, error) {
	cmd += "\n"

	_, err := p.cont.Stdin.Write([]byte(cmd))
	if err != nil {
		return "", err
	}

	if !p.scanner.Scan() {
		return "", convertContainerErrToPlayerErr(p.scanner.Err())
	}

	return p.scanner.Text(), nil
}

func (p *DockerPlayer) RetrieveField(conf field.Configuration) (field.Field, error) {
	r, err := p.cont.ReadFile("/tmp/field.txt")

	if err != nil {
		return nil, convertContainerErrToPlayerErr(err)
	}

	f := field.NewShipField(0)

	ships := field.ParseShips(r)
	err = f.Load(conf, ships)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (p *DockerPlayer) Close() error {
	return p.cont.Close()
}
