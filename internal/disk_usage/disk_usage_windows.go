package disk_usage

import (
	"syscall"
	"unsafe"
)

// Loaded on first call. kernel32.dll is a KnownDLL,
// Windows always loads it from the system directory.
var procGetDiskFreeSpaceExW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func diskUsage(path string) (Usage, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Usage{}, err
	}
	// BOOL GetDiskFreeSpaceExW(LPCWSTR lpDirectoryName,
	//     PULARGE_INTEGER lpFreeBytesAvailableToCaller,  // respects user disk quotas
	//     PULARGE_INTEGER lpTotalNumberOfBytes,          // respects user disk quotas
	//     PULARGE_INTEGER lpTotalNumberOfFreeBytes);     // whole disk
	var freeToCaller, total, totalFree uint64
	ok, _, callErr := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&totalFree)))
	if ok == 0 {
		// Call always returns non-nil error, it is meaningful only if the function failed
		return Usage{}, callErr
	}
	return Usage{
		Total: int64(total),
		// with quotas totalFree can be > total
		Used: max(int64(total)-int64(totalFree), 0),
		Free: int64(freeToCaller),
	}, nil
}
