// Copyright 2015 Aleksandr Demakin. All rights reserved.

package ipc

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"bitbucket.org/avd/go-ipc/internal/test"
)

const (
	shmProgName  = "./internal/test/shared_memory/main.go"
	fifoProgName = "./internal/test/fifo/main.go"
	syncProgName = "./internal/test/sync/main.go"
)

type testAppResult struct {
	output string
	err    error
}

// Shared memory test program
func argsForShmCreateCommand(name string, size int64) []string {
	return []string{shmProgName, "-object=" + name, "create", fmt.Sprintf("%d", size)}
}

func argsForShmDestroyCommand(name string) []string {
	return []string{shmProgName, "-object=" + name, "destroy"}
}

func argsForShmReadCommand(name string, offset int64, lenght int) []string {
	return []string{shmProgName, "-object=" + name, "read", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", lenght)}
}

func argsForShmTestCommand(name string, offset int64, data []byte) []string {
	strBytes := ipc_test.BytesToString(data)
	return []string{shmProgName, "-object=" + name, "test", fmt.Sprintf("%d", offset), strBytes}
}

func argsForShmWriteCommand(name string, offset int64, data []byte) []string {
	strBytes := ipc_test.BytesToString(data)
	return []string{shmProgName, "-object=" + name, "write", fmt.Sprintf("%d", offset), strBytes}
}

// FIFO memory test program
func argsForFifoCreateCommand(name string) []string {
	return []string{fifoProgName, "-object=" + name, "create"}
}

func argsForFifoDestroyCommand(name string) []string {
	return []string{fifoProgName, "-object=" + name, "destroy"}
}

func argsForFifoReadCommand(name string, nonblock bool, lenght int) []string {
	return []string{fifoProgName, "-object=" + name, "-nonblock=" + boolStr(nonblock), "read", fmt.Sprintf("%d", lenght)}
}

func argsForFifoTestCommand(name string, nonblock bool, data []byte) []string {
	strBytes := ipc_test.BytesToString(data)
	return []string{fifoProgName, "-object=" + name, "-nonblock=" + boolStr(nonblock), "test", strBytes}
}

func argsForFifoWriteCommand(name string, nonblock bool, data []byte) []string {
	strBytes := ipc_test.BytesToString(data)
	return []string{fifoProgName, "-object=" + name, "-nonblock=" + boolStr(nonblock), "write", strBytes}
}

// Sync test program

func argsForSyncCreateCommand(name, t string) []string {
	return []string{syncProgName, "-object=" + name, "-type=" + t, "create"}
}

func argsForSyncDestroyCommand(name string) []string {
	return []string{syncProgName, "-object=" + name, "destroy"}
}

func argsForSyncInc64Command(name, t string, jobs int, shm_name string, n int) []string {
	return []string{
		syncProgName,
		"-object=" + name,
		"-type=" + t,
		"-jobs=" + strconv.Itoa(jobs),
		"inc64",
		shm_name,
		strconv.Itoa(n),
	}
}

func argsForSyncTestCommand(name, t string, jobs int, shm_name string, n int, data []byte, log string) []string {
	return []string{
		syncProgName,
		"-object=" + name,
		"-type=" + t,
		"-jobs=" + strconv.Itoa(jobs),
		"-log=" + log,
		"test",
		shm_name,
		strconv.Itoa(n),
		ipc_test.BytesToString(data),
	}
}

func boolStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func runTestApp(args []string, killChan <-chan bool) (result testAppResult) {
	args = append([]string{"run"}, args...)
	cmd := exec.Command("go", args...)
	buff := bytes.NewBuffer(nil)
	cmd.Stderr = buff
	cmd.Stdout = buff
	if err := cmd.Start(); err != nil {
		result.err = err
		return
	}
	if killChan != nil {
		go func() {
			if kill, ok := <-killChan; kill && ok {
				if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
					cmd.Process.Kill()
				}
			}
		}()
	}
	fmt.Printf("started new process [%d]\n", cmd.Process.Pid)
	result.err = cmd.Wait()
	if result.err != nil {
		if exiterr, ok := result.err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				result.err = fmt.Errorf("%v, status code = %d", result.err, status)
			}
		}
	} else {
		if !cmd.ProcessState.Success() {
			result.err = fmt.Errorf("process has exited with an error")
		}
	}
	result.output = buff.String()
	return
}

func runTestAppAsync(args []string, killChan <-chan bool) <-chan testAppResult {
	ch := make(chan testAppResult, 1)
	go func() {
		ch <- runTestApp(args, killChan)
	}()
	return ch
}

func waitForFunc(f func(), d time.Duration) bool {
	ch := make(chan bool, 1)
	go func() {
		f()
		ch <- true
	}()
	select {
	case <-ch:
		return true
	case <-time.After(d):
		return false
	}
}

func waitForAppResultChan(ch <-chan testAppResult, d time.Duration) (testAppResult, bool) {
	select {
	case value := <-ch:
		return value, true
	case <-time.After(d):
		return testAppResult{}, false
	}
}
