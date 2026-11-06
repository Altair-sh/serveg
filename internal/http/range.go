package http

import (
	"strconv"
	"strings"
)

type RangeResult int

const (
	// no header, invalid syntax or multiple ranges: serve the full body with 200
	RangeNone RangeResult = iota
	// serve bytes [start, end] (inclusive) with 206
	RangeOK
	// respond with 416
	RangeUnsatisfiable
)

// ParseRange parses a "Range" header value for a resource of the given size.
// Only a single byte range is supported: "bytes=a-b", "bytes=a-", "bytes=-n".
// Anything else is ignored (RangeNone), which RFC 9110 §14.2 allows.
func ParseRange(header string, size int64) (start, end int64, r RangeResult) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, 0, RangeNone
	}
	spec, ok := strings.CutPrefix(header, "bytes=")
	if !ok || strings.ContainsRune(spec, ',') {
		return 0, 0, RangeNone
	}
	firstStr, lastStr, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, RangeNone
	}
	firstStr = strings.TrimSpace(firstStr)
	lastStr = strings.TrimSpace(lastStr)

	if firstStr == "" {
		// suffix range: last n bytes
		n, ok := parseNonNegative(lastStr)
		if !ok {
			return 0, 0, RangeNone
		}
		if n == 0 || size == 0 {
			return 0, 0, RangeUnsatisfiable
		}
		return max(size-n, 0), size - 1, RangeOK
	}

	first, ok := parseNonNegative(firstStr)
	if !ok {
		return 0, 0, RangeNone
	}
	last := size - 1
	if lastStr != "" {
		last, ok = parseNonNegative(lastStr)
		if !ok || last < first {
			return 0, 0, RangeNone
		}
	}
	if first >= size {
		return 0, 0, RangeUnsatisfiable
	}
	return first, min(last, size-1), RangeOK
}

func parseNonNegative(s string) (int64, bool) {
	n, err := strconv.ParseUint(s, 10, 63)
	return int64(n), err == nil
}
