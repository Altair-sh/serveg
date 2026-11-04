package html

import sh "serveg/internal/string_helper"

const DateTimeTZ = "2006-01-02 15:04:05 UTC-07"

func htmlEscape(s string) string {
	sb := sh.StringBuilder{Buf: make([]byte, 0, len(s))}
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch b {
		case '&':
			sb.AddS("&amp;")
		case '<':
			sb.AddS("&lt;")
		case '>':
			sb.AddS("&gt;")
		case '"':
			sb.AddS("&quot;")
		case '\'':
			sb.AddS("&#39;")
		default:
			sb.AddB(b)
		}
	}
	return sb.StringUnsafe()
}
