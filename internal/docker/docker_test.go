package docker_test

import (
	"archive/tar"
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrsobakin/itmournament/internal/docker"
)

func mockFileServer(t testing.TB) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		path = strings.TrimSuffix(path, ".tar")
		path = "./testdata/" + path

		tw := tar.NewWriter(w)
		defer tw.Close()

		tw.AddFS(os.DirFS(path))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func buildContainer(t testing.TB, name string) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)

	url := mockFileServer(t)

	ctx := context.Background()

	b, err := docker.NewSubmissionBuilder(cli, ctx, "")
	require.NoError(t, err)

	res := b.Build(ctx, docker.Source{
		Src: url + "/" + name + ".tar",
	})

	require.NoError(t, res.Err, "container should build")

	return res.ImageId
}

func getRunner(t testing.TB, limits docker.Limits) *docker.SubmissionRunner {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)

	return docker.NewSubmissionRunner(cli, limits)
}

func getContainer(t testing.TB, ctx context.Context, name string, limits docker.Limits) *docker.SubmissionContainer {
	imageId := buildContainer(t, name)

	cont, err := getRunner(t, limits).CreateSubmissionContainer(ctx, imageId)
	require.NoError(t, err)

	return cont
}

func runContainer(t testing.TB, path string, limits docker.Limits) (int64, string) {
	cont := getContainer(t, context.Background(), path, limits)
	defer cont.Close()

	assert.NoError(t, cont.Start())

	buf := new(strings.Builder)
	io.Copy(buf, cont.Stdout)

	result := cont.Wait()
	assert.NoError(t, result.Err)

	return result.ExitCode, strings.TrimSpace(buf.String())
}

func Test_Limits_VCPUs(t *testing.T) {
	limitsFull := docker.Limits{
		VCPUs:  1,
		Memory: 128 * 1024 * 1024,
	}
	limitsHalf := limitsFull
	limitsHalf.VCPUs = 0.5

	ecFull, outFull := runContainer(t, "counter", limitsFull)
	valFull, err := strconv.Atoi(outFull)
	require.NoError(t, err)
	assert.Equal(t, int64(0), ecFull)
	t.Logf("Full: %d", valFull)

	ecHalf, outHalf := runContainer(t, "counter", limitsHalf)
	valHalf, err := strconv.Atoi(outHalf)
	require.NoError(t, err)
	assert.Equal(t, int64(0), ecHalf)
	t.Logf("Half: %d", valHalf)

	ratio := float64(valHalf) / float64(valFull)
	t.Logf("Ratio: %f", ratio)
	assert.LessOrEqual(t, ratio, 0.5)
}

func Test_Limits_Memory(t *testing.T) {
	limitsEnough := docker.Limits{
		VCPUs:  1,
		Memory: (128 + 6) * 1024 * 1024,
	}
	limitsInsuff := limitsEnough
	limitsInsuff.Memory = 128 * 1024 * 1024

	ecEnough, outEnough := runContainer(t, "memory", limitsEnough)
	assert.Equal(t, "ok", outEnough)
	assert.Equal(t, int64(0), ecEnough)

	ecInsuff, outInsuff := runContainer(t, "memory", limitsInsuff)
	assert.Equal(t, "", outInsuff)
	assert.Equal(t, int64(137), ecInsuff)
}

func Test_ReadFile(t *testing.T) {
	cont := getContainer(t, context.Background(), "file", docker.Limits{})
	defer cont.Close()

	require.NoError(t, cont.Start())

	reader, err := cont.ReadFile("/tmp/file.txt")
	assert.NoError(t, err)

	content := new(strings.Builder)
	io.Copy(content, reader)
	reader.Close()

	cont.Wait()

	assert.Equal(t, "test data", content.String())
}

func Test_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("timeout")

	cont := getContainer(t, ctx, "counter", docker.Limits{})
	require.NoError(t, cont.Start())

	time.Sleep(100 * time.Millisecond)
	cancel(cause)

	result := cont.Wait()
	assert.ErrorIs(t, result.Err, cause)
	assert.Equal(t, int64(-1), result.ExitCode)
}

func Benchmark_Throughput_Lines(b *testing.B) {
	cont := getContainer(b, context.Background(), "echo", docker.Limits{})
	require.NoError(b, cont.Start())
	defer cont.Close()

	scanner := bufio.NewScanner(cont.Stdout)
	scanner.Split(bufio.ScanLines)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cont.Stdin.Write([]byte("a\n"))
		assert.True(b, scanner.Scan())
		assert.Equal(b, "a", scanner.Text())
	}
	b.StopTimer()
}
