package main

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/mrsobakin/itmournament/internal/docker"
	"github.com/mrsobakin/itmournament/internal/game"
)

func tryBindParams(ctx *gin.Context, obj any) (ok bool) {
	if err := ctx.BindJSON(&obj); err != nil {
		ctx.JSON(422, map[string]any{
			"error":   ErrBadFormat,
			"details": err.Error(),
		})
		return false
	}
	return true
}

type dockerFactory struct {
	runner  *docker.SubmissionRunner
	imageId string
}

func NewDockerFactory(runner *docker.SubmissionRunner, imageId string) *dockerFactory {
	return &dockerFactory{
		runner,
		imageId,
	}
}

func (d *dockerFactory) NewPlayer(ctx context.Context) game.Player {
	p, err := docker.NewDockerPlayer(d.runner, ctx, d.imageId)
	if err != nil {
		panic(err)
	}

	return p
}
