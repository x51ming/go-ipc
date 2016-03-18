// Copyright 2015 Aleksandr Demakin. All rights reserved.

package ipc

import (
	"fmt"
	"syscall"
	"unsafe"

	"bitbucket.org/avd/go-ipc/internal/allocator"

	"golang.org/x/sys/unix"
)

const (
	cSIGEV_SIGNAL      = 0
	cSIGEV_NONE        = 1
	cSIGEV_THREAD      = 2
	cNOTIFY_COOKIE_LEN = 32
)

func initMqNotifications(ch chan<- int) (int, error) {
	notifySocketFd, err := syscall.Socket(syscall.AF_NETLINK,
		syscall.SOCK_RAW|syscall.SOCK_CLOEXEC,
		syscall.NETLINK_ROUTE)
	if err != nil {
		return -1, err
	}
	go func() {
		var data [cNOTIFY_COOKIE_LEN]byte
		for {
			n, _, err := syscall.Recvfrom(notifySocketFd, data[:], syscall.MSG_NOSIGNAL|syscall.MSG_WAITALL)
			if n == cNOTIFY_COOKIE_LEN && err == nil {
				p := unsafe.Pointer(&data[0])
				ndata := (*notify_data)(p)
				ch <- ndata.mq_id
				use(p)
			} else {
				return
			}
		}
	}()
	return notifySocketFd, nil
}

// syscalls

type notify_data struct {
	mq_id   int
	padding [cNOTIFY_COOKIE_LEN - unsafe.Sizeof(int(0))]byte
}

type sigval struct { /* Data passed with notification */
	sigval_ptr uintptr /* A pointer-sized value to match the union size in syscall */
}

type sigevent struct {
	sigev_value             sigval
	sigev_signo             int32
	sigev_notify            int32
	sigev_notify_function   uintptr
	sigev_notify_attributes uintptr
	padding                 [8]int32 // 8 is the maximum padding size
}

func mq_open(name string, flags int, mode uint32, attrs *MqAttr) (int, error) {
	nameBytes, err := syscall.BytePtrFromString(name)
	if err != nil {
		return -1, err
	}
	bytes := unsafe.Pointer(nameBytes)
	attrsP := unsafe.Pointer(attrs)
	id, _, err := syscall.Syscall6(unix.SYS_MQ_OPEN,
		uintptr(bytes),
		uintptr(flags),
		uintptr(mode),
		uintptr(attrsP),
		0,
		0)
	use(bytes)
	use(attrsP)
	if err != syscall.Errno(0) {
		return -1, err
	}
	return int(id), nil
}

func mq_timedsend(id int, data []byte, prio int, timeout *unix.Timespec) error {
	rawData := allocator.ByteSliceData(data)
	timeoutPtr := unsafe.Pointer(timeout)
	_, _, err := syscall.Syscall6(unix.SYS_MQ_TIMEDSEND,
		uintptr(id),
		uintptr(rawData),
		uintptr(len(data)),
		uintptr(prio),
		uintptr(timeoutPtr),
		0)
	use(rawData)
	use(timeoutPtr)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func dump(ptr unsafe.Pointer, off, size int) {
	s := uintptr(ptr) + uintptr(off)
	for i := 0; i < size; i++ {
		l := s + uintptr(i)
		bPtr := (*byte)(unsafe.Pointer(l))
		fmt.Printf("%X", *bPtr)
	}
	fmt.Println()
}

func mq_timedreceive(id int, data []byte, prio *int, timeout *unix.Timespec) (int, int, error) {
	rawData := allocator.ByteSliceData(data)
	timeoutPtr := unsafe.Pointer(timeout)
	prioPtr := unsafe.Pointer(prio)
	dump(rawData, -16, 32)
	msgSize, maxMsgSize, err := syscall.Syscall6(unix.SYS_MQ_TIMEDRECEIVE,
		uintptr(id),
		uintptr(rawData),
		uintptr(len(data)),
		uintptr(prioPtr),
		uintptr(timeoutPtr),
		0)
	dump(rawData, -16, 32)
	use(rawData)
	use(timeoutPtr)
	use(prioPtr)
	if err != syscall.Errno(0) {
		return 0, 0, err
	}
	return int(msgSize), int(maxMsgSize), nil
}

func mq_notify(id int, event *sigevent) error {
	eventPtr := unsafe.Pointer(event)
	_, _, err := syscall.Syscall(unix.SYS_MQ_NOTIFY, uintptr(id), uintptr(eventPtr), uintptr(0))
	use(eventPtr)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func mq_getsetattr(id int, attrs, oldAttrs *MqAttr) error {
	attrsPtr := unsafe.Pointer(attrs)
	oldAttrsPtr := unsafe.Pointer(oldAttrs)
	_, _, err := syscall.Syscall(unix.SYS_MQ_GETSETATTR,
		uintptr(id),
		uintptr(attrsPtr),
		uintptr(oldAttrsPtr))
	use(attrsPtr)
	use(oldAttrsPtr)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func mq_unlink(name string) error {
	nameBytes, err := syscall.BytePtrFromString(name)
	if err != nil {
		return err
	}
	bytes := unsafe.Pointer(nameBytes)
	_, _, err = syscall.Syscall(unix.SYS_MQ_UNLINK, uintptr(bytes), uintptr(0), uintptr(0))
	use(bytes)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
