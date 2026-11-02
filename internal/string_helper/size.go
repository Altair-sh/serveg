package string_helper

import (
	"math"
	"strconv"
	"strings"
)

// Format byte size to kilobytes, megabytes, etc.
//
//	1023       -> "1023"
//	1048576    -> "1M"
//	4778151117 -> "4.45G"
//	1023*1024  -> "1M"
func HumanBytes(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	v := float64(n)
	i := 0
	for i < len(units)-1 && math.Round(math.Abs(v)*100)/100 >= 1024 {
		v /= 1024
		i++
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s + units[i]
}
