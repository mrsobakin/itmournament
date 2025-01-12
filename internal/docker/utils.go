package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session/secrets"
	"github.com/opencontainers/go-digest"
)

type Limits struct {
	// Memory in bytes
	Memory int64
	VCPUs  float64
}

type dockerStdoutReader struct {
	left  uint32
	muxed *bufio.Reader
}

func newDockerStdoutReader(r *bufio.Reader) *dockerStdoutReader {
	return &dockerStdoutReader{
		left:  0,
		muxed: r,
	}
}

func (r *dockerStdoutReader) Read(p []byte) (int, error) {
	if r.left == 0 {
		header := make([]byte, 8)

		n, err := r.muxed.Read(header)

		if err != nil {
			return n, err
		}

		if n != 8 {
			return 0, nil
		}

		// Skip non-stdout
		if header[0] != 1 {
			return 0, nil
		}

		r.left = binary.BigEndian.Uint32(header[4:])
	}

	if r.left > uint32(cap(p)) {
		r.left -= uint32(cap(p))
	} else {
		r.left = 0
	}

	return r.muxed.Read(p)
}

type gitAuthTokenProvider struct {
	token string
}

func (p *gitAuthTokenProvider) GetSecret(ctx context.Context, id string) ([]byte, error) {
	if id != "GIT_AUTH_TOKEN" {
		return nil, secrets.ErrNotFound
	}
	return []byte(p.token), nil
}

type untarReader struct {
	foundHeader bool
	close       func() error
	tar         *tar.Reader
}

func newUntarReader(r io.ReadCloser) *untarReader {
	return &untarReader{
		foundHeader: false,
		close:       r.Close,
		tar:         tar.NewReader(r),
	}
}

func (r *untarReader) Close() error {
	return r.close()
}

func (r *untarReader) Read(buf []byte) (int, error) {
	if !r.foundHeader {
		r.tar.Next()
		r.foundHeader = true
	}

	return r.tar.Read(buf)
}

type readerWithContainerError struct {
	inner io.Reader
	cont  *SubmissionContainer
}

func readerInjectContainerError(r io.Reader, cont *SubmissionContainer) io.Reader {
	return &readerWithContainerError{
		inner: r,
		cont:  cont,
	}
}

func (r *readerWithContainerError) Read(p []byte) (int, error) {
	n, err := r.inner.Read(p)

	if errors.Is(err, io.EOF) {
		result := r.cont.Wait()
		err = &ErrorTerminated{result}
	}

	return n, err
}

func updateLogsFromStep(ctx context.Context, w io.Writer, ch <-chan *client.SolveStatus, predicate func(*client.Vertex) bool) error {
	var keepFromVertex digest.Digest

	for {
		select {
		case s, ok := <-ch:
			if !ok {
				return nil
			}

			for _, v := range s.Vertexes {
				if predicate(v) {
					keepFromVertex = v.Digest
				}
			}

			for _, l := range s.Logs {
				if l.Vertex == keepFromVertex {
					if _, err := w.Write(l.Data); err != nil {
						return err
					}
				}
			}

		case <-ctx.Done():
			return nil
		}
	}
}
