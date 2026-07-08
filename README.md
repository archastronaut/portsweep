# portsweep

A concurrent TCP port scanner in Go, using a bounded goroutine worker pool.

## Build

```sh
go build ./cmd/portsweep
```

## Usage

```sh
portsweep -host scanme.nmap.org -ports 1-1024
portsweep -host 192.168.1.1 -ports 22,80,443,8000-8100 -workers 200 -banner
```

| Flag | Default | Description |
|------|---------|-------------|
| `-host` | — | target host or IP (required) |
| `-ports` | `1-1024` | ports/ranges, e.g. `22,80,443,8000-8100` |
| `-workers` | `100` | concurrent workers |
| `-timeout` | `2s` | per-port dial timeout |
| `-banner` | `false` | best-effort banner grab on open ports |

> Only scan hosts you own or have explicit permission to test.
