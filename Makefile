BINARY := kanban
DIST := dist

.PHONY: run build build-cross build-all test tidy

run:
	go run .

build:
	go build -o dist/$(BINARY) .

# Cross-compile for one platform, e.g.:
#   make build-cross GOOS=linux GOARCH=amd64
#   make build-cross GOOS=windows GOARCH=amd64
build-cross:
	@mkdir -p $(DIST)
	@ext=""; \
	if [ "$(GOOS)" = "windows" ]; then ext=".exe"; fi; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(DIST)/$(BINARY)-$(GOOS)-$(GOARCH)$$ext .

build-all:
	$(MAKE) build-cross GOOS=linux GOARCH=amd64
	$(MAKE) build-cross GOOS=linux GOARCH=arm64
	$(MAKE) build-cross GOOS=darwin GOARCH=amd64
	$(MAKE) build-cross GOOS=darwin GOARCH=arm64
	$(MAKE) build-cross GOOS=windows GOARCH=amd64

test:
	go test ./...

tidy:
	go mod tidy
