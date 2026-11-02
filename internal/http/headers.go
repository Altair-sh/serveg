package http

import (
	"net/url"
	sh "serveg/internal/string_helper"
	"strings"
)

// ContentDisposition builds a Content-Disposition header value (RFC 6266).
// dispositionType is "inline" or "attachment".
// filename is an ASCII fallback for old clients,
// filename* is the exact UTF-8 name.
//
//	ContentDisposition("inline", "отчёт 1.txt") ->
//	inline; filename="_____ 1.txt"; filename*=UTF-8''%D0%BE%D1%82%D1%87%D1%91%D1%82%201.txt
func ContentDisposition(dispositionType, fileName string) string {
	var sb sh.StringBuilder
	sb.Grow(len(dispositionType) + len(fileName)*4 + 32)
	sb.AddS(dispositionType)
	sb.AddS(`; filename="`).AddS(AsciiFallbackName(fileName)).AddB('"')
	sb.AddS(`; filename*=UTF-8''`).AddS(PercentEncodeHeaderValue(fileName))
	return sb.StringUnsafe()
}

// percent-encodes characters that are not allowed in urls
func PercentEcodeUrl(p string) string {
	u := url.URL{Path: p}
	return u.EscapedPath()
}

// PercentEncodeHeaderValue percent-encodes characters that are not allowed in header values
func PercentEncodeHeaderValue(s string) string {
	const hexDigits = "0123456789ABCDEF"
	var sb sh.StringBuilder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isAttrChar(c) {
			sb.AddB(c)
		} else {
			sb.AddB('%').AddB(hexDigits[c>>4]).AddB(hexDigits[c&0x0F])
		}
	}
	return sb.StringUnsafe()
}

// attr-char = ALPHA / DIGIT / "!" / "#" / "$" / "&" / "+" / "-" / "."
// / "^" / "_" / "`" / "|" / "~"
func isAttrChar(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("!#$&+-.^_`|~", c) >= 0
}

// AsciiFallbackName replaces each character that can't be safely put
// in a quoted-string with '_'. One '_' per character, not per byte.
func AsciiFallbackName(s string) string {
	var sb sh.StringBuilder
	sb.Grow(len(s))
	// range over string iterates runes, invalid UTF-8 gives utf8.RuneError
	for _, r := range s {
		if r < 0x20 || r >= 0x7F || r == '"' || r == '\\' || r == '%' {
			r = '_'
		}
		sb.AddB(byte(r))
	}
	return sb.StringUnsafe()
}
