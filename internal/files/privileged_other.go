//go:build !linux

package files

import "os"

func readPrivilegedMetadata(*os.File) (privilegedMetadata, error) {
	return privilegedMetadata{}, ErrUnsupported
}
func restorePrivilegedMetadata(*os.File, privilegedMetadata) error { return ErrUnsupported }
