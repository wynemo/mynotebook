package notebook

import "golang.org/x/sys/unix"

// Move never replaces a destination created concurrently by another application.
func Move(src, dst string) error { return unix.RenamexNp(src, dst, unix.RENAME_EXCL) }
