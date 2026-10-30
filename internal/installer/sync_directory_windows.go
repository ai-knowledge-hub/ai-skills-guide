//go:build windows

package installer

// Windows does not support syncing an opened directory handle through
// os.File.Sync. File contents are synced before rename and the transaction
// journal remains available for recovery after process interruption.
func syncDirectory(string) error {
	return nil
}
