package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sync/semaphore"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/mrsobakin/itmournament/internal/docker"
)

func InitDockerThings(limits docker.Limits) (*docker.SubmissionBuilder, *docker.SubmissionRunner, error) {
	ctx := context.Background()

	var err error
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		return nil, nil, err
	}

	token, ok := os.LookupEnv("GIT_AUTH_TOKEN")

	if !ok {
		return nil, nil, fmt.Errorf("GIT_AUTH_TOKEN is not set")
	}

	builder, err := docker.NewSubmissionBuilder(cli, ctx, token)
	runner := docker.NewSubmissionRunner(cli, limits)

	return builder, runner, err
}

func NewServer() *server {
	limits := docker.Limits{
		Memory: 70 * 1024 * 1024,
		VCPUs:  1,
	}

	builder, runner, err := InitDockerThings(limits)
	if err != nil {
		panic(err)
	}

	nCPU := runtime.NumCPU() * 2

	return &server{
		builder: builder,
		runner:  runner,
		jobs:    semaphore.NewWeighted(int64(nCPU)),
	}
}

func main() {
	router := gin.Default()

	s := NewServer()

	s.RegisterEndpoints(router)

	addr := "127.0.0.1:4239"
	if len(os.Args) >= 2 {
		addr = os.Args[1]
	}

	fmt.Println(router.Run(addr))
}
