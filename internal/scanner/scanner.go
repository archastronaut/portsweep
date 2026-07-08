// Package scanner provides a concurrent TCP connect-scanner built on a
// bounded goroutine worker pool.
package scanner

import (
	"context"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Result describes the outcome of probing a single port.
type Result struct {
	Port   int
	Open   bool
	Banner string // best-effort service banner, empty if none read
}

// Config controls a scan.
type Config struct {
	Host    string
	Ports   []int
	Workers int
	Timeout time.Duration
	Grab    bool // attempt a best-effort banner read on open ports
}

// Scan probes every port in cfg.Ports against cfg.Host using a pool of
// cfg.Workers goroutines. It returns the open ports sorted ascending.
// The scan stops early if ctx is cancelled.
func Scan(ctx context.Context, cfg Config) []Result {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}

	ports := make(chan int)
	results := make(chan Result)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range ports {
				results <- probe(ctx, cfg, port)
			}
		}()
	}

	// Feed ports, honoring cancellation.
	go func() {
		defer close(ports)
		for _, p := range cfg.Ports {
			select {
			case <-ctx.Done():
				return
			case ports <- p:
			}
		}
	}()

	// Close results once all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	var open []Result
	for r := range results {
		if r.Open {
			open = append(open, r)
		}
	}
	sort.Slice(open, func(i, j int) bool { return open[i].Port < open[j].Port })
	return open
}

// probe attempts a single TCP connect against host:port.
func probe(ctx context.Context, cfg Config, port int) Result {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	d := net.Dialer{Timeout: cfg.Timeout}

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Result{Port: port, Open: false}
	}
	defer conn.Close()

	res := Result{Port: port, Open: true}
	if cfg.Grab {
		res.Banner = grabBanner(conn, cfg.Timeout)
	}
	return res
}

// grabBanner does a best-effort read of a service banner. Many services
// (SSH, SMTP, FTP) announce themselves on connect; HTTP does not, so an
// empty banner is normal and not an error.
func grabBanner(conn net.Conn, timeout time.Duration) string {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}
	return sanitize(buf[:n])
}

// sanitize trims the banner to its first line and strips control bytes so
// terminal output stays readable.
func sanitize(b []byte) string {
	out := make([]rune, 0, len(b))
	for _, r := range string(b) {
		if r == '\n' || r == '\r' {
			break
		}
		if r == '\t' || (r >= 0x20 && r != 0x7f) {
			out = append(out, r)
		}
	}
	return string(out)
}
