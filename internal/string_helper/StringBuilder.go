package string_helper

import (
	"io"
	"slices"
	"strconv"
	"unicode/utf8"
)

type StringBuilder struct {
	// allocate it with size of estimated output
	Buf []byte
}

// compile-time interface checks
var (
	_ io.Writer       = (*StringBuilder)(nil)
	_ io.StringWriter = (*StringBuilder)(nil)
	_ io.ByteWriter   = (*StringBuilder)(nil)
)

// Write implements [io.Writer]. It never returns an error.
func (b *StringBuilder) Write(p []byte) (int, error) {
	b.Buf = append(b.Buf, p...)
	return len(p), nil
}

// WriteString implements [io.StringWriter]. It never returns an error.
func (b *StringBuilder) WriteString(s string) (int, error) {
	b.Buf = append(b.Buf, s...)
	return len(s), nil
}

// WriteByte implements [io.ByteWriter]. It never returns an error.
func (b *StringBuilder) WriteByte(c byte) error {
	b.Buf = append(b.Buf, c)
	return nil
}

// Reset empties the builder but keeps the allocated capacity for reuse.
func (b *StringBuilder) Reset() { b.Buf = b.Buf[:0] }

// Grow ensures there is room for at least n more bytes
func (b *StringBuilder) Grow(n int) { b.Buf = slices.Grow(b.Buf, n) }

func (b *StringBuilder) Len() int { return len(b.Buf) }

func (b *StringBuilder) AddS(v string) *StringBuilder {
	b.Buf = append(b.Buf, v...)
	return b
}

func (b *StringBuilder) AddB(v byte) *StringBuilder {
	b.Buf = append(b.Buf, v)
	return b
}

func (b *StringBuilder) AddR(v rune) *StringBuilder {
	b.Buf = utf8.AppendRune(b.Buf, v)
	return b
}

func (b *StringBuilder) AddBool(v bool) *StringBuilder {
	b.Buf = strconv.AppendBool(b.Buf, v)
	return b
}

// numberBase: 2 <= base <= 36
func (b *StringBuilder) AddIExt(v int64, numberBase int) *StringBuilder {
	b.Buf = strconv.AppendInt(b.Buf, v, numberBase)
	return b
}
func (b *StringBuilder) AddI(v int64) *StringBuilder   { return b.AddIExt(v, 10) }
func (b *StringBuilder) AddI32(v int32) *StringBuilder { return b.AddI(int64(v)) }
func (b *StringBuilder) AddI16(v int16) *StringBuilder { return b.AddI(int64(v)) }
func (b *StringBuilder) AddI8(v int8) *StringBuilder   { return b.AddI(int64(v)) }

// numberBase: 2 <= base <= 36
func (b *StringBuilder) AddUExt(v uint64, numberBase int) *StringBuilder {
	b.Buf = strconv.AppendUint(b.Buf, v, numberBase)
	return b
}
func (b *StringBuilder) AddU(v uint64) *StringBuilder   { return b.AddUExt(v, 10) }
func (b *StringBuilder) AddU32(v uint32) *StringBuilder { return b.AddU(uint64(v)) }
func (b *StringBuilder) AddU16(v uint16) *StringBuilder { return b.AddU(uint64(v)) }
func (b *StringBuilder) AddU8(v uint8) *StringBuilder   { return b.AddU(uint64(v)) }

// fmtChar: see [strconv.FormatFloat]
// precision: number of digits after . (-1 for max precision)
func (b *StringBuilder) AddF32Ext(v float32, fmtChar byte, precision int) *StringBuilder {
	b.Buf = strconv.AppendFloat(b.Buf, float64(v), fmtChar, precision, 32)
	return b
}
func (b *StringBuilder) AddF32(v float32) *StringBuilder {
	return b.AddF32Ext(v, 'g', -1)
}

// fmtChar: see [strconv.FormatFloat]
// precision: number of digits after . (-1 for max precision)
func (b *StringBuilder) AddF64Ext(v float64, fmtChar byte, precision int) *StringBuilder {
	b.Buf = strconv.AppendFloat(b.Buf, v, fmtChar, precision, 64)
	return b
}
func (b *StringBuilder) AddF64(v float64) *StringBuilder {
	return b.AddF64Ext(v, 'g', -1)
}

// creates copy of buffer
func (b *StringBuilder) String() string {
	return string(b.Buf)
}

// doesn't copy buffer,
// so you shouldn't modify b.Buf before using the string
func (b *StringBuilder) StringUnsafe() string {
	return UnsafeBytesToStr(b.Buf)
}
