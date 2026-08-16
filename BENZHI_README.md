# LAN Clipboard Sync Package

This package contains a Go command-line application that discovers devices on a local network over UDP and sends text clipboard data over authenticated HTTP.

## Container build

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh lan-clipboard-sync linux/arm64
./build_benzhi_docker.sh lan-clipboard-sync linux/amd64
```

The image retains the Go 1.22 toolchain for evaluation and development.

## Validation inside the container

```bash
go build ./...
go test ./...
go vet ./...
```

## Local execution

```bash
go build ./cmd/lansync
./lansync -name "My Device" -key 'a-long-private-shared-key' serve
```

For full application usage, platform clipboard requirements, and manual send commands, see `README.md`.
