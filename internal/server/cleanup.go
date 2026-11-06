package server

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PUT writes into filePath+UploadingSuffix and renames it when the body is received
const UploadingSuffix = ".uploading"

// RemoveUploadingFiles deletes UploadingSuffix files left by interrupted uploads
// (server crash, kill -9). Call it before the server starts, while nothing is uploading.
// Errors are logged, not returned: a share that can't be cleaned is still usable.
func (shares Shares) RemoveUploadingFiles() {
	for _, sh := range shares {
		// WalkDir doesn't follow symlinks, so it doesn't leave the share
		_ = filepath.WalkDir(sh.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				confLogger.Warn("cleanup: ", err)
				return nil // skip unreadable entry, continue walking
			}
			if !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), UploadingSuffix) {
				return nil
			}
			err = os.Remove(path)
			switch {
			case err == nil:
				confLogger.Info("cleanup: removed ", path)
			// nested shares (shared/ and shared/private) walk the same files twice
			case errors.Is(err, fs.ErrNotExist):
			default:
				confLogger.Warn("cleanup: ", err)
			}
			return nil
		})
	}
}
