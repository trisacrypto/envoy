package gds

import "errors"

var (
	ErrAlreadyConnected = errors.New("already connected to directory, cannot overide dialer")
	ErrNotConnected     = errors.New("not connected to directory service")
)
