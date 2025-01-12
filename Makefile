TARGET = github.com/mrsobakin/itmournament/cmd/server
OUT = itmournament

all: build test

internal/docker/.cache/buildctx.tar: internal/docker/Dockerfile
	mkdir -p internal/docker/.cache/
	tar cf $@ -C internal/docker Dockerfile

.PHONY: build
build: internal/docker/.cache/buildctx.tar
	mkdir -p bin/
	go build -o bin/$(OUT) $(TARGET)

.PHONY: run
run:
	go run $(TARGET) $(ARGS)

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test ./... -test.v

.PHONY: coverage
coverage:
	mkdir -p .cache
	go test -cover -coverprofile .cache/cover.out ./...

.PHONY: coverage-html
coverage-html: coverage
	go tool cover -html=.cache/cover.out

.PHONY: clean
clean:
	-rm -rf bin/
	-find -type d -name '.cache' -exec rm -r {} +
