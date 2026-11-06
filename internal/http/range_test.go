package http

import "testing"

func TestParseRange(t *testing.T) {
	const size = 1000
	tests := []struct {
		header     string
		start, end int64
		result     RangeResult
	}{
		{"", 0, 0, RangeNone},
		{"bytes=0-", 0, 999, RangeOK},
		{"bytes=500-", 500, 999, RangeOK},
		{"bytes=0-499", 0, 499, RangeOK},
		{"bytes=999-999", 999, 999, RangeOK},
		{"bytes=-100", 900, 999, RangeOK},
		{"bytes=-5000", 0, 999, RangeOK},
		{"bytes=900-5000", 900, 999, RangeOK},
		{" bytes= 10 - 20 ", 10, 20, RangeOK},
		{"bytes=1000-", 0, 0, RangeUnsatisfiable},
		{"bytes=1000-2000", 0, 0, RangeUnsatisfiable},
		{"bytes=-0", 0, 0, RangeUnsatisfiable},
		{"bytes=5-1", 0, 0, RangeNone},
		{"bytes=0-1,5-9", 0, 0, RangeNone},
		{"items=0-1", 0, 0, RangeNone},
		{"bytes=abc-", 0, 0, RangeNone},
		{"bytes=+5-", 0, 0, RangeNone},
		{"bytes=--5", 0, 0, RangeNone},
		{"bytes=5", 0, 0, RangeNone},
		{"bytes=-", 0, 0, RangeNone},
		{"bytes=99999999999999999999-", 0, 0, RangeNone},
		{"bytes=9223372036854775807-", 0, 0, RangeUnsatisfiable}, // MaxInt64
		{"bytes=9223372036854775808-", 0, 0, RangeNone},          // MaxInt64+1
		{"bytes=0-9223372036854775808", 0, 0, RangeNone},
		{"bytes=1_0-", 0, 0, RangeNone},
		{"bytes=-+5", 0, 0, RangeNone},
	}
	for _, tt := range tests {
		start, end, r := ParseRange(tt.header, size)
		if r != tt.result || start != tt.start || end != tt.end {
			t.Errorf("ParseRange(%q) = (%d, %d, %d), want (%d, %d, %d)",
				tt.header, start, end, r, tt.start, tt.end, tt.result)
		}
	}

	// empty file: any range is unsatisfiable
	if _, _, r := ParseRange("bytes=0-", 0); r != RangeUnsatisfiable {
		t.Errorf("empty file bytes=0-: got %d, want RangeUnsatisfiable", r)
	}
	if _, _, r := ParseRange("bytes=-10", 0); r != RangeUnsatisfiable {
		t.Errorf("empty file bytes=-10: got %d, want RangeUnsatisfiable", r)
	}
}
