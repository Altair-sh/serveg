//go:build !(linux || darwin || freebsd || windows)

// Fallback, so the program still builds on other OSes.

package disk_usage

func diskUsage(path string) (Usage, error) {
	return Usage{}, ErrNotSupported
}
