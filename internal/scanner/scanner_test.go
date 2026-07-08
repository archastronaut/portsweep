package scanner

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    []int
		wantErr bool
	}{
		{name: "single", spec: "80", want: []int{80}},
		{name: "list", spec: "22,80,443", want: []int{22, 80, 443}},
		{name: "range", spec: "20-23", want: []int{20, 21, 22, 23}},
		{name: "mixed and deduped", spec: "80, 22-23, 80", want: []int{22, 23, 80}},
		{name: "whitespace", spec: " 80 , 443 ", want: []int{80, 443}},
		{name: "empty", spec: "", wantErr: true},
		{name: "zero", spec: "0", wantErr: true},
		{name: "too high", spec: "70000", wantErr: true},
		{name: "inverted range", spec: "100-10", wantErr: true},
		{name: "garbage", spec: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePorts(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePorts(%q) err = %v, wantErr %v", tt.spec, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePorts(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

// TestScanFindsOpenPort spins up a real listener on an OS-assigned port and
// confirms the scanner reports it open while a neighbouring closed port is not.
func TestScanFindsOpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Accept and immediately close, so the port reads as open.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	openPort := ln.Addr().(*net.TCPAddr).Port

	// Find a port that is (almost certainly) closed by opening then freeing one.
	tmp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tmp: %v", err)
	}
	closedPort := tmp.Addr().(*net.TCPAddr).Port
	tmp.Close()

	results := Scan(context.Background(), Config{
		Host:    "127.0.0.1",
		Ports:   []int{openPort, closedPort},
		Workers: 4,
		Timeout: time.Second,
	})

	if len(results) != 1 || results[0].Port != openPort {
		t.Fatalf("expected only port %d open, got %+v", openPort, results)
	}
}

func TestScanCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before we start

	// Use a non-routable address so any dial that slips through would hang;
	// cancellation must make Scan return promptly regardless.
	done := make(chan []Result, 1)
	go func() {
		done <- Scan(ctx, Config{
			Host:    "10.255.255.1",
			Ports:   []int{80, 443, 8080},
			Workers: 2,
			Timeout: 30 * time.Second,
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Scan did not honour cancelled context")
	}
}
