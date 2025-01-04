// Copyright 2016 Aleksandr Demakin. All rights reserved.

package sync

// #include <pthread.h>
import "C"
import (
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

func gettid() (uint32, error) {
	port := C.pthread_mach_thread_np(C.pthread_self())
	if port == 0 {
		return 0, errors.New("failed to get mach port of the current thread")
	}
	return uint32(port), nil
}

func killThread(port uint32) error {
	thread := C.pthread_from_mach_thread_np(C.mach_port_t(port))
	if thread == nil {
		return errors.New("failed to get pthread_t")
	}
	r := C.pthread_kill(thread, C.int(syscall.SIGUSR2))
	if r != 0 {
		return unix.Errno(r)
	}
	return nil
}
