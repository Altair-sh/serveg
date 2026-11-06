package disk_usage

import "errors"

var ErrNotSupported = errors.New("disk_usage: not supported on this OS")

// Used + Free can be less than Total: space reserved for root/admin
// is not free for this process, but not used either.
type Usage struct {
	// size of the file system (available to this user, if there are quotas)
	Total int64
	// Total minus all free space, including reserved
	Used int64
	// available to this process
	Free int64
}

// DiskUsage returns usage of the file system that contains path.
func DiskUsage(path string) (Usage, error) {
	return diskUsage(path)
}
