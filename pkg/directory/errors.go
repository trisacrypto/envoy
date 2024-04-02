package directory

import "errors"

var (
	ErrSyncAlreadyRunning = errors.New("directory sync is already running")
	ErrSyncNotRunning     = errors.New("directory sync is not running")
)
