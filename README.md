# LAN Clipboard Sync

A small Go command-line tool for sharing text clipboard content between devices on the same LAN. Devices discover one another through UDP broadcast, then transfer clipboard text over HTTP. A shared key authenticates both discovery and content messages.

## Requirements

- Go 1.22 or newer. The application itself uses only the Go standard library.
- macOS: `pbcopy` and `pbpaste` are built in.
- Linux: install `xclip` for the optional clipboard read/write integration. Explicit `-text` sends work without it.
- Hosts must allow inbound TCP on port 8743 and UDP broadcasts on port 8742 (or configured alternatives). Some guest/corporate networks block UDP broadcast; use `send -to` with a known address in that case.

## Build

```sh
go build ./cmd/lansync
```

## Start

Choose a sufficiently long secret and use the same secret on every device.

```sh
# Start discovery and receive messages
./lansync -name "Alice Mac" -key 'replace-with-a-long-private-key' serve

# Include local clipboard polling and broadcast each changed text value
./lansync -name "Alice Mac" -key 'replace-with-a-long-private-key' -watch serve
```

The default data directory is the operating system configuration directory under `lansync`. Use `-data-dir ./data` to keep history within a chosen directory.

## Send and inspect

```sh
# List discovered devices (waits briefly for broadcasts)
./lansync -key 'replace-with-a-long-private-key' devices

# Send current clipboard to a selected receiver
./lansync -key 'replace-with-a-long-private-key' -to 192.168.1.50:8743 send

# Send a supplied string, useful on servers without a clipboard
./lansync -key 'replace-with-a-long-private-key' -to 192.168.1.50:8743 -text 'hello' send

# View locally recorded receive/send history
./lansync -key 'replace-with-a-long-private-key' history
```

Discovery identifies devices by a stable `-id`; set a distinct value when hosts share the same hostname. `-watch` only sends changes made after startup, avoiding a startup broadcast. Receiver errors and unreachable devices are logged without stopping the service. Recent message IDs are retained for 24 hours, preventing repeated deliveries caused by retries.

## Test

```sh
go test ./...
go vet ./...
```
