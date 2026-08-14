BINARY_NAME=shield
DAEMON_NAME=shieldd
VERSION=0.1.0
LDFLAGS=-ldflags="-s -w"

PLATFORMS=linux darwin
ARCHITECTURES=amd64 arm64

release:
	@echo "Cleaning build dir..."
	rm -rf target/dist/
	mkdir -p target/dist/
	@for os in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(LDFLAGS) -o target/dist/$(BINARY_NAME) cmd/shield/main.go; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(LDFLAGS) -o target/dist/$(DAEMON_NAME) cmd/shieldd/main.go; \
			tar -czvf target/dist/$(BINARY_NAME)_$(VERSION)_$${os}_$${arch}.tar.gz -C target/dist $(BINARY_NAME) $(DAEMON_NAME); \
			rm -f target/dist/$(BINARY_NAME) target/dist/$(DAEMON_NAME); \
		done \
	done
	@cd target/dist && sha256sum *.tar.gz > sha256sums.txt
	@echo "BUILD OK."
