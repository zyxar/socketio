GO          = go
GOARCH     := amd64

all: $(shell $(GO) env GOOS)

build-%:
	$(eval $@_OS := $*)
	env GOOS=$($@_OS) GOARCH=$(GOARCH) $(GO) build

linux: build-linux

darwin: build-darwin

install-%:
	$(eval $@_OS := $*)
	env GOOS=$($@_OS) GOARCH=$(GOARCH) $(GO) install

install: install-$(shell $(GO) env GOOS)

test:
	$(GO) test -v -race -cover

bench:
	@$(GO) test -v -bench=. -run=X
