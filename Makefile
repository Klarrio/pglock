SOURCEDIR = .
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY = pglock

VERSION := $(shell cat VERSION)
BUILD_TIME := $(shell date +%FT%T%z)
GITREV := $(shell git rev-parse HEAD)
GOVERSION := $(shell go version)

LDFLAGS := -ldflags '-X "main.GitRev=${GITREV}" -X "main.Version=${VERSION}" -X "main.BuildTime=${BUILD_TIME}" -X "main.GoVersion=${GOVERSION}"'

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY}

.PHONY: clean_release
clean_release:
	@echo "Cleaning up dist/.."
	rm dist/*.tar.gz
	rm dist/${BINARY}_*

.PHONY: release
release: clean_release
	@mkdir -p dist/
	CGO_ENABLED=0 gox -osarch="darwin/amd64 linux/amd64 freebsd/amd64" ${LDFLAGS} --output "dist/${BINARY}_{{.OS}}_{{.Arch}}"

	tar -czvf dist/${BINARY}-linux-${VERSION}.tar.gz dist/${BINARY}_linux_amd64
	tar -czvf dist/${BINARY}-macos-${VERSION}.tar.gz dist/${BINARY}_darwin_amd64
	tar -czvf dist/${BINARY}-freebsd-${VERSION}.tar.gz dist/${BINARY}_freebsd_amd64

	echo "Built artifacts:" dist/*.tar.gz
