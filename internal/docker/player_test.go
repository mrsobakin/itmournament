package docker_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrsobakin/itmournament/internal/docker"
	"github.com/mrsobakin/itmournament/internal/game"
	"github.com/mrsobakin/itmournament/internal/game/field"
)

func getPlayer(t testing.TB, memoryLimit int64) (game.Player, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	imageId := buildContainer(t, "game")

	runner := getRunner(t, docker.Limits{
		Memory: memoryLimit,
	})

	player, err := docker.NewDockerPlayer(runner, ctx, imageId)
	require.NoError(t, err)

	return player, cancel
}

func Test_MemoryLimit(t *testing.T) {
	player, cancel := getPlayer(t, 120*1024*1024)
	defer cancel()
	defer player.Close()

	for i := 1; i <= 3; i++ {
		str := strconv.Itoa(i)

		resp, err := player.SendCommand("echo " + str)
		assert.NoError(t, err)
		assert.Equal(t, str, resp)
	}

	_, err := player.SendCommand("echo 4")
	assert.ErrorIs(t, err, &game.ErrorTerminated{Reason: game.ReasonMemoryLimit})
}

func Test_Exit(t *testing.T) {
	player, cancel := getPlayer(t, 500*1024*1024)
	defer cancel()
	defer player.Close()

	for i := 1; i <= 5; i++ {
		str := strconv.Itoa(i)

		resp, err := player.SendCommand("echo " + str)
		assert.NoError(t, err)
		assert.Equal(t, str, resp)
	}

	_, err := player.SendCommand("echo 6")
	assert.ErrorIs(t, err, &game.ErrorTerminated{Reason: game.ReasonNormal})
}

func Test_RuntimeError(t *testing.T) {
	player, cancel := getPlayer(t, 500*1024*1024)
	defer cancel()
	defer player.Close()

	_, err := player.SendCommand("echo asd")
	assert.ErrorIs(t, err, &game.ErrorTerminated{Reason: game.ReasonRuntimeError})
}

func Test_GetField(t *testing.T) {
	player, cancel := getPlayer(t, 500*1024*1024)
	defer cancel()
	defer player.Close()

	resp, err := player.SendCommand("field /tmp/field.txt")
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	f, err := player.RetrieveField(field.Configuration{
		W:     10,
		H:     10,
		Sizes: [4]int64{1, 1, 1, 1},
	})
	require.NoError(t, err)

	assert.Equal(t, field.Kill, f.Shoot(0, 0))

	assert.Equal(t, field.Hit, f.Shoot(0, 2))
	assert.Equal(t, field.Kill, f.Shoot(1, 2))

	assert.Equal(t, field.Hit, f.Shoot(0, 4))
	assert.Equal(t, field.Hit, f.Shoot(1, 4))
	assert.Equal(t, field.Kill, f.Shoot(2, 4))

	assert.Equal(t, field.Hit, f.Shoot(0, 6))
	assert.Equal(t, field.Hit, f.Shoot(1, 6))
	assert.Equal(t, field.Hit, f.Shoot(2, 6))
	assert.Equal(t, field.Kill, f.Shoot(3, 6))

	assert.True(t, f.AllDead())
}

func Test_GetField_MemoryLimit(t *testing.T) {
	player, cancel := getPlayer(t, 150*1024*1024)
	defer cancel()
	defer player.Close()

	resp, err := player.SendCommand("field /tmp/field.txt")
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	time.Sleep(time.Second)

	f, err := player.RetrieveField(field.Configuration{
		W:     10,
		H:     10,
		Sizes: [4]int64{1, 1, 1, 1},
	})
	assert.ErrorIs(t, err, &game.ErrorTerminated{Reason: game.ReasonMemoryLimit})
	assert.Nil(t, f)
}
