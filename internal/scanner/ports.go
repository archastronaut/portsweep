package scanner

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ParsePorts expands a port specification into a sorted, de-duplicated
// slice. The spec is a comma-separated list of single ports and inclusive
// ranges, e.g. "22,80,443,8000-8100". Whitespace is ignored.
func ParsePorts(spec string) ([]int, error) {
	seen := make(map[int]struct{})
	for _, field := range strings.Split(spec, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		lo, hi, err := parseField(field)
		if err != nil {
			return nil, err
		}
		for p := lo; p <= hi; p++ {
			seen[p] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("no ports parsed from %q", spec)
	}

	ports := make([]int, 0, len(seen))
	for p := range seen {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports, nil
}

func parseField(field string) (lo, hi int, err error) {
	if before, after, found := strings.Cut(field, "-"); found {
		lo, err = parsePort(before)
		if err != nil {
			return 0, 0, err
		}
		hi, err = parsePort(after)
		if err != nil {
			return 0, 0, err
		}
		if lo > hi {
			return 0, 0, fmt.Errorf("range %q is inverted", field)
		}
		return lo, hi, nil
	}
	p, err := parsePort(field)
	return p, p, err
}

func parsePort(s string) (int, error) {
	p, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid port %q", s)
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("port %d out of range 1-65535", p)
	}
	return p, nil
}
