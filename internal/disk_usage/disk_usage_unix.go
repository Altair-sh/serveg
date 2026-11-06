//go:build linux || darwin || freebsd

// "_unix" is not a special file name suffix for Go, so the OS list is in the //go:build line.
// Not "unix": openbsd and netbsd have different Statfs_t or no syscall.Statfs.

package disk_usage

import "syscall"

func diskUsage(path string) (Usage, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Usage{}, err
	}
	// Field types differ between OSes, int64() works for all of them.
	bsize := int64(st.Bsize)
	return Usage{
		Total: int64(st.Blocks) * bsize,
		// Bfree: all free blocks, including reserved for root
		Used: (int64(st.Blocks) - int64(st.Bfree)) * bsize,
		// Bavail: free blocks available to unprivileged users
		Free: int64(st.Bavail) * bsize,
	}, nil
}
