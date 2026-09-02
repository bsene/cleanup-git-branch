.PHONY: all build test vet clean install

BINARY := cleanup-git-branch

all: vet test build

build:
	go build -o $(BINARY) ./cmd/cleanup-git-branch

test:
	go test -race ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/cleanup-git-branch
