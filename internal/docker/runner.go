package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type SubmissionRunner struct {
	cli    *client.Client
	limits Limits
}

func NewSubmissionRunner(cli *client.Client, limits Limits) *SubmissionRunner {
	return &SubmissionRunner{
		cli,
		limits,
	}
}

type RunResult struct {
	ExitCode int64
	Err      error
}

type ErrorTerminated struct {
	Result RunResult
}

func (e *ErrorTerminated) Error() string {
	return fmt.Sprintf("container terminated with [ec: %d, err: %s]", e.Result.ExitCode, e.Result.Err)
}

type SubmissionContainer struct {
	runner *SubmissionRunner
	ctx    context.Context
	id     string

	started   atomic.Bool
	closed    atomic.Bool
	wg        sync.WaitGroup
	runResult RunResult

	Stdin  io.Writer
	Stdout io.Reader
}

func (r *SubmissionRunner) CreateSubmissionContainer(ctx context.Context, imageName string) (*SubmissionContainer, error) {
	stopTimeout := 1

	resp, err := r.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:           imageName,
			NetworkDisabled: true,
			AttachStderr:    false,
			AttachStdin:     true,
			AttachStdout:    true,
			Tty:             false,
			OpenStdin:       true,
			StopTimeout:     &stopTimeout,
		},
		&container.HostConfig{
			AutoRemove: true,
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyDisabled,
			},
			LogConfig: container.LogConfig{
				Type: "none",
			},
			Resources: container.Resources{
				NanoCPUs: int64(r.limits.VCPUs * 1e9),
				Memory:   r.limits.Memory,
			},
		},
		nil,
		nil,
		"",
	)

	if err != nil {
		return nil, err
	}

	waiter, err := r.cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})

	if err != nil {
		return nil, err
	}

	cont := &SubmissionContainer{
		runner: r,
		ctx:    ctx,
		id:     resp.ID,
		Stdin:  waiter.Conn,
	}

	cont.Stdout = readerInjectContainerError(newDockerStdoutReader(waiter.Reader), cont)

	return cont, nil
}

func (c *SubmissionContainer) Start() error {
	if !c.started.CompareAndSwap(false, true) {
		panic("same container started multiple times")
	}

	err := c.runner.cli.ContainerStart(c.ctx, c.id, container.StartOptions{})
	if err != nil {
		c.runResult = RunResult{0, err}
		c.removeContainer()
		return err
	}

	c.wg.Add(1)
	go func() {
		statusChan, errChan := c.runner.cli.ContainerWait(c.ctx, c.id, container.WaitConditionNotRunning)

		select {
		case <-c.ctx.Done():
			c.removeContainer()
			c.runResult = RunResult{-1, context.Cause(c.ctx)}
		case err := <-errChan:
			c.runResult = RunResult{-1, err}
		case status := <-statusChan:
			c.runResult = RunResult{status.StatusCode, nil}
		}
		c.closed.Store(true)

		c.wg.Done()
	}()

	return err
}

func (c *SubmissionContainer) ReadFile(path string) (io.ReadCloser, error) {
	reader, _, err := c.runner.cli.CopyFromContainer(c.ctx, c.id, path)

	if err != nil {
		if !client.IsErrNotFound(err) {
			return nil, err
		}

		// There is no way to unwrap saving message, so we'll use this dirty hack
		if !strings.HasPrefix(err.Error(), "Error response from daemon: No such container: ") {
			return nil, err
		}

		result := c.Wait()
		return nil, &ErrorTerminated{result}
	}

	untar := newUntarReader(reader)
	return untar, nil
}

func (c *SubmissionContainer) Wait() RunResult {
	if !c.started.Load() {
		return RunResult{0, fmt.Errorf("container was never started")}
	}

	c.wg.Wait()

	return c.runResult
}

func (c *SubmissionContainer) removeContainer() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	// Use background context because container should be deleted regardless
	return c.runner.cli.ContainerRemove(context.Background(), c.id, container.RemoveOptions{
		Force: true,
	})
}

func (c *SubmissionContainer) Close() error {
	err := c.removeContainer()
	if err != nil {
		return err
	}

	c.wg.Wait()

	return nil
}
