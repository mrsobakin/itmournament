package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/semaphore"

	"github.com/mrsobakin/itmournament/internal/docker"
	"github.com/mrsobakin/itmournament/internal/judge"
)

const (
	BuildTimeout  time.Duration = 100 * time.Minute
	PlayerTimeout time.Duration = 2 * time.Minute
	GlobalTimeout time.Duration = 7 * time.Minute
)

const (
	ErrBadRepo   string = "bad_repo"
	ErrBadFormat string = "bad_format"
	ErrUnknown   string = "unknown"
	ErrTimeout   string = "timeout"
)

var (
	errBuildTimeout error = errors.New("build timeout")
)

type server struct {
	builder *docker.SubmissionBuilder
	runner  *docker.SubmissionRunner
	jobs    *semaphore.Weighted
}

func (s *server) handleBuild(c *gin.Context) {
	var params struct {
		Repo string `json:"repo" binding:"required"`
		Ref  string `json:"ref" binding:"required"`
	}

	if !tryBindParams(c, &params) {
		return
	}

	s.jobs.Acquire(c, 1)
	defer s.jobs.Release(1)

	timeoutCtx, cancel := context.WithTimeoutCause(context.Background(), BuildTimeout, errBuildTimeout)
	defer cancel()

	result := s.builder.Build(timeoutCtx, docker.Source{
		Repo: params.Repo,
		Ref:  params.Ref,
	})

	if result.Err == nil {
		c.JSON(200, map[string]any{
			"image_id": result.ImageId,
			"logs":     result.Logs,
		})
		return
	}

	var err string
	var errCode int

	if strings.HasPrefix(result.Err.Error(), "failed to solve: failed to load cache key: error fetching default branch for repository https://github.com/.git:") {
		errCode = 400
		err = ErrBadRepo
	} else if errors.Is(result.Err, errBuildTimeout) {
		errCode = 408
		err = ErrTimeout
	} else {
		errCode = 400
		err = ErrUnknown
	}

	c.JSON(errCode, map[string]any{
		"error":   err,
		"details": result.Err.Error(),
		"logs":    result.Logs,
	})
}

func (s *server) handleMatch(c *gin.Context) {
	var params struct {
		MasterImageId string `json:"master_image_id" binding:"required"`
		SlaveImageId  string `json:"slave_image_id" binding:"required"`
	}

	if !tryBindParams(c, &params) {
		return
	}

	s.jobs.Acquire(c, 2)
	defer s.jobs.Release(2)

	j := judge.Judge{
		PlayerTimeout: PlayerTimeout,
		GlobalTimeout: GlobalTimeout,
	}

	verdict := j.Judge(
		c.Request.Context(),
		NewDockerFactory(s.runner, params.MasterImageId),
		NewDockerFactory(s.runner, params.SlaveImageId),
	)

	c.JSON(200, verdict)
}

func (s *server) RegisterEndpoints(e *gin.Engine) {
	e.POST("/build", s.handleBuild)
	e.POST("/run_match", s.handleMatch)
}
