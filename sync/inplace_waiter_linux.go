// Copyright 2016 Aleksandr Demakin. All rights reserved.

package sync

import (
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"bitbucket.org/avd/go-ipc/internal/common"
)

type inplaceWaiter uint32

func newInplaceWaiter(addr unsafe.Pointer) *inplaceWaiter {
	return (*inplaceWaiter)(addr)
}

func (w *inplaceWaiter) addr() *uint32 {
	return (*uint32)(unsafe.Pointer(w))
}

func (w *inplaceWaiter) add(value int) {
	atomic.AddUint32(w.addr(), uint32(value))
}

func (w *inplaceWaiter) wait(value uint32, timeout time.Duration) error {
	err := FutexWait(unsafe.Pointer(w), value, timeout, 0)
	if common.SyscallErrHasCode(err, syscall.EWOULDBLOCK) {
		return nil
	}
	return err
}

func (w *inplaceWaiter) wake(count uint32) (int, error) {
	return FutexWake(unsafe.Pointer(w), count, 0)
}

func (w *inplaceWaiter) wakeAll() (int, error) {
	return w.wake(cFutexWakeAll)
}