// Copyright 2015 Aleksandr Demakin. All rights reserved.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package ipc

const (
	// O_NONBLOCK flag makes some ipc operations non-blocking
	O_NONBLOCK = 0x00000040
)
