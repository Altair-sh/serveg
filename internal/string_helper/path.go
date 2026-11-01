package string_helper

import (
	"strings"
)

// Returns path cleaned from "./", "/./", "//"
// or empty string if path is not safe.
// Path is considered unsafe if it:
// - contains ".." segments
// - ends with '.'
// - contains `\:*?"<>|`
// - contains windows-restricted names COM, CON, ...
func ResolveSafePath(decodedPath string) string {
	var sb StringBuilder
	sb.Grow(len(decodedPath))
	var seg string
	for rest := decodedPath; rest != ""; {
		seg, rest = nextSegment(rest)
		if seg == "" || seg == "." {
			continue
		}
		if !isSafeSegment(seg) {
			return ""
		}
		sb.AddB('/')
		sb.AddS(seg)
	}
	if sb.Len() == 0 {
		return "/"
	}
	return sb.StringUnsafe()
}

// nextSegment splits off the first '/'-separated segment of s.
func nextSegment(s string) (seg, rest string) {
	seg, rest, _ = strings.Cut(s, "/")
	return
}

// isSafeSegment reports whether a single non-empty path segment is safe to
// use as a file name.
func isSafeSegment(seg string) bool {
	if seg == ".." {
		return false
	}
	// Windows strips trailing dots "a.txt." would open "a.txt"
	if last := seg[len(seg)-1]; last == '.' {
		return false
	}
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		if c < 0x20 || c == 0x7F {
			return false
		}
		// invalid in Windows file names, ':' also selects NTFS streams
		if strings.IndexByte(`\:*?"<>|`, c) >= 0 {
			return false
		}
	}
	return !isReservedWindowsName(seg)
}

// isReservedWindowsName reports whether seg names a Windows device:
// "CON", "con.txt", "COM1.tar.gz", ...
func isReservedWindowsName(seg string) bool {
	base := seg
	if i := strings.IndexByte(seg, '.'); i >= 0 {
		base = seg[:i]
	}
	switch len(base) {
	case 3:
		return asciiEqualUpper(base, "CON") || asciiEqualUpper(base, "PRN") ||
			asciiEqualUpper(base, "AUX") || asciiEqualUpper(base, "NUL")
	case 4:
		prefix := base[:3]
		return (asciiEqualUpper(prefix, "COM") || asciiEqualUpper(prefix, "LPT")) &&
			'0' <= base[3] && base[3] <= '9'
	}
	return false
}

// asciiUpper upper-cases ASCII letters only (like std::toupper in the "C" locale).
func asciiUpper(c byte) byte {
	if 'a' <= c && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

// asciiEqualUpper compares s with upper (which must be upper-case ASCII), ignoring
// ASCII case. Unlike strings.EqualFold it does no Unicode folding, so e.g. the
// Kelvin sign 'K' (U+212A) does not match 'K'.
func asciiEqualUpper(s, upper string) bool {
	if len(s) != len(upper) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if asciiUpper(s[i]) != upper[i] {
			return false
		}
	}
	return true
}
