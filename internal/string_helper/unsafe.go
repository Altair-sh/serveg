package string_helper

import "unsafe"

func UnsafeGetStrBytes(s string) []byte {
	pointr := unsafe.StringData(s)
	return unsafe.Slice(pointr, len(s))
}

func UnsafeBytesToStr(b []byte) string {
	pointr := unsafe.SliceData(b)
	return unsafe.String(pointr, len(b))
}
