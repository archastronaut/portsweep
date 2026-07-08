// Command portsweep is a concurrent TCP connect port scanner.
//
// It probes a host across a set of ports using a bounded pool of worker
// goroutines and reports which ports accept connections.
//
// Only scan hosts you own or have explicit permission to test.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/archastronaut/portsweep/internal/scanner"
)

func main() {
	host := flag.String("host", "", "target host or IP (required)")
	portSpec := flag.String("ports", "1-1024", "ports to scan, e.g. 22,80,443,8000-8100")
	workers := flag.Int("workers", 100, "number of concurrent workers")
	timeout := flag.Duration("timeout", 2*time.Second, "per-port dial timeout")
	grab := flag.Bool("banner", false, "attempt best-effort banner grab on open ports")
	flag.Parse()

	if *host == "" {
		fmt.Fprintln(os.Stderr, "error: -host is required")
		flag.Usage()
		os.Exit(2)
	}

	ports, err := scanner.ParsePorts(*portSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	// Cancel the scan cleanly on Ctrl-C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("scanning %s: %d ports, %d workers, %s timeout\n",
		*host, len(ports), *workers, *timeout)

	start := time.Now()
	open := scanner.Scan(ctx, scanner.Config{
		Host:    *host,
		Ports:   ports,
		Workers: *workers,
		Timeout: *timeout,
		Grab:    *grab,
	})
	elapsed := time.Since(start).Round(time.Millisecond)

	for _, r := range open {
		if r.Banner != "" {
			fmt.Printf("%d/tcp open  %s\n", r.Port, r.Banner)
		} else {
			fmt.Printf("%d/tcp open\n", r.Port)
		}
	}
	fmt.Printf("\n%d open / %d scanned in %s\n", len(open), len(ports), elapsed)
}
