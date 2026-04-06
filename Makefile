BINARY_NAME=shield
VERSION=1.0.0
LDFLAGS=-ldflags="-s -w"

PLATFORMS=linux darwin
ARCHITECTURES=amd64 arm64

release:
	@echo "Cleaning build dir..."
	rm -rf dist/
	mkdir -p dist/
	@for os in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME) main.go; \
			tar -czvf dist/$(BINARY_NAME)_$(VERSION)_$${os}_$${arch}.tar.gz -C dist $(BINARY_NAME); \
			rm dist/$(BINARY_NAME); \
		done \
	done
	@cd dist && sha256sum *.tar.gz > sha256sums.txt
	@echo "BUILD OK."