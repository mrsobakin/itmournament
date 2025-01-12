package docker

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"strings"

	"github.com/docker/buildx/driver"
	_ "github.com/docker/buildx/driver/docker"
	dockerclient "github.com/docker/docker/client"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/session/upload/uploadprovider"

	// "github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"
)

//go:embed .cache/buildctx.tar
var buildCtxTarBytes []byte

func getDockerBuildkitClient(ctx context.Context, client *dockerclient.Client) (*client.Client, error) {
	driverFactory, err := driver.GetFactory("docker", false)
	if err != nil {
		return nil, err
	}

	dockerHandle, err := driver.GetDriver(ctx, driverFactory, driver.InitConfig{
		DockerAPI: client,
	})

	if err != nil {
		return nil, err
	}

	return dockerHandle.Client(ctx)
}

type SubmissionBuilder struct {
	buildkitClient *client.Client
	tokenProvider  *gitAuthTokenProvider
}

func NewSubmissionBuilder(cli *dockerclient.Client, ctx context.Context, token string) (*SubmissionBuilder, error) {
	buildkitClient, err := getDockerBuildkitClient(ctx, cli)
	if err != nil {
		return nil, err
	}

	return &SubmissionBuilder{
		buildkitClient: buildkitClient,
		tokenProvider: &gitAuthTokenProvider{
			token,
		},
	}, nil
}

type BuildResult struct {
	ImageId string
	Logs    string
	Err     error
}

type Source struct {
	Repo string
	Ref  string
	Src  string
}

func (b *SubmissionBuilder) Build(ctx context.Context, src Source) BuildResult {
	up := uploadprovider.New()
	buildCtx := up.Add(io.NopCloser(bytes.NewReader(buildCtxTarBytes)))

	opts := client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				Type: "moby",
				Attrs: map[string]string{
					"name": "submission",
				},
			},
		},
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"context":        buildCtx,
			"build-arg:repo": src.Repo,
			"build-arg:ref":  src.Ref,
			"build-arg:src":  src.Src,
		},
		Session: []session.Attachable{
			secretsprovider.NewSecretProvider(b.tokenProvider),
			up,
		},
	}

	var result BuildResult
	var logBytes bytes.Buffer

	ch := make(chan *client.SolveStatus)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		resp, err := b.buildkitClient.Build(ctx, opts, "", dockerfile.Build, ch)
		if err == nil {
			imageId := resp.ExporterResponse["containerimage.digest"]
			result.ImageId = imageId
		}

		return err
	})

	eg.Go(func() error {
		return updateLogsFromStep(ctx, &logBytes, ch, func(v *client.Vertex) bool {
			// TODO: it seems like there is no way to change vertex name
			// in docker file. So, kludges it is.
			return strings.HasPrefix(v.Name, "[builder 4/4] RUN --network=none ")
		})
	})

	result.Err = eg.Wait()
	result.Logs = logBytes.String()

	return result
}
