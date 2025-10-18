package gui

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func expandIPPattern(input string) ([]string, error) {
	expr := strings.TrimSpace(input)
	if expr == "" {
		return nil, errors.New("please enter an IP address or range")
	}

	if strings.Contains(expr, "/") {
		return expandCIDR(expr)
	}

	if start, end, ok := parseDashRange(expr); ok {
		return expandIPRange(start, end)
	}

	if !strings.ContainsAny(expr, "*-") {
		if net.ParseIP(expr) == nil {
			return nil, fmt.Errorf("invalid IP address: %s", expr)
		}
		return []string{expr}, nil
	}

	parts := strings.Split(expr, ".")
	if len(parts) != 4 {
		return nil, errors.New("expected four octets in IP range")
	}

	ranges := make([][]int, 4)
	for i, part := range parts {
		values, err := expandOctet(part)
		if err != nil {
			return nil, fmt.Errorf("octet %d: %w", i+1, err)
		}
		ranges[i] = values
	}

	total := 1
	for _, r := range ranges {
		total *= len(r)
		if total > deviceScanLimit {
			return nil, fmt.Errorf("range expands to %d addresses; limit is %d", total, deviceScanLimit)
		}
	}

	ips := make([]string, 0, total)
	for _, a := range ranges[0] {
		for _, b := range ranges[1] {
			for _, c := range ranges[2] {
				for _, d := range ranges[3] {
					ips = append(ips, fmt.Sprintf("%d.%d.%d.%d", a, b, c, d))
				}
			}
		}
	}
	return ips, nil
}

func expandOctet(part string) ([]int, error) {
	trimmed := strings.TrimSpace(part)
	if trimmed == "" {
		return nil, errors.New("empty octet")
	}
	if trimmed == "*" {
		values := make([]int, 256)
		for i := 0; i < 256; i++ {
			values[i] = i
		}
		return values, nil
	}
	if strings.Contains(trimmed, "-") {
		pieces := strings.Split(trimmed, "-")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("invalid range: %s", trimmed)
		}
		start, err := parseOctetValue(pieces[0])
		if err != nil {
			return nil, err
		}
		end, err := parseOctetValue(pieces[1])
		if err != nil {
			return nil, err
		}
		if start > end {
			return nil, fmt.Errorf("range start greater than end: %s", trimmed)
		}
		values := make([]int, 0, end-start+1)
		for i := start; i <= end; i++ {
			values = append(values, i)
		}
		return values, nil
	}
	value, err := parseOctetValue(trimmed)
	if err != nil {
		return nil, err
	}
	return []int{value}, nil
}

func parseOctetValue(v string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 || n > 255 {
		return 0, fmt.Errorf("invalid octet value: %s", v)
	}
	return n, nil
}

func expandCIDR(expr string) ([]string, error) {
	ip, network, err := net.ParseCIDR(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %w", err)
	}

	v4 := ip.To4()
	if v4 == nil {
		return nil, errors.New("CIDR notation must be IPv4")
	}

	ones, bits := network.Mask.Size()
	if bits != 32 {
		return nil, errors.New("CIDR notation must be IPv4")
	}

	hostBits := bits - ones
	if hostBits < 0 {
		return nil, errors.New("invalid CIDR mask")
	}

	networkVal, err := ipToUint32(v4)
	if err != nil {
		return nil, err
	}

	var start, end uint32
	switch {
	case hostBits == 0:
		start = networkVal
		end = networkVal
	case hostBits == 1:
		start = networkVal
		end = networkVal + 1
	default:
		blockSize := uint32(1 << hostBits)
		start = networkVal + 1
		end = networkVal + blockSize - 2
	}

	if end < start {
		end = start
	}

	count := end - start + 1
	if count == 0 {
		return nil, errors.New("CIDR range produced no addresses")
	}
	if count > uint32(deviceScanLimit) {
		return nil, fmt.Errorf("CIDR expands to %d addresses; limit is %d", count, deviceScanLimit)
	}

	ips := make([]string, 0, count)
	for val := start; val <= end; val++ {
		ips = append(ips, uint32ToIP(val).String())
	}
	return ips, nil
}

func expandIPRange(startStr, endStr string) ([]string, error) {
	startIP := net.ParseIP(startStr)
	endIP := net.ParseIP(endStr)
	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP range: %s - %s", startStr, endStr)
	}

	startVal, err := ipToUint32(startIP)
	if err != nil {
		return nil, err
	}
	endVal, err := ipToUint32(endIP)
	if err != nil {
		return nil, err
	}

	if endVal < startVal {
		return nil, errors.New("end IP must not be lower than start IP")
	}

	count := endVal - startVal + 1
	if count > uint32(deviceScanLimit) {
		return nil, fmt.Errorf("range expands to %d addresses; limit is %d", count, deviceScanLimit)
	}

	ips := make([]string, 0, count)
	for val := startVal; val <= endVal; val++ {
		ips = append(ips, uint32ToIP(val).String())
	}
	return ips, nil
}

func parseDashRange(expr string) (string, string, bool) {
	if !strings.Contains(expr, "-") {
		return "", "", false
	}
	parts := strings.SplitN(expr, "-", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if start == "" || end == "" {
		return "", "", false
	}
	if net.ParseIP(start) == nil || net.ParseIP(end) == nil {
		return "", "", false
	}
	return start, end, true
}

func ipToUint32(ip net.IP) (uint32, error) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, fmt.Errorf("only IPv4 addresses are supported: %s", ip.String())
	}
	return uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3]), nil
}

func uint32ToIP(value uint32) net.IP {
	return net.IPv4(
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	).To4()
}
