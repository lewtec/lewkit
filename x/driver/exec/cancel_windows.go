//go:build windows

package exec

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// jobs holds a Job Object per started process so cancel kills the
// shell and the children that would otherwise keep its pipes open.
var jobs sync.Map // pid → windows.Handle

func prepareCancel(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

func trackProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	err = windows.AssignProcessToJobObject(job, proc)
	_ = windows.CloseHandle(proc)
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	jobs.Store(cmd.Process.Pid, job)
}

func finishCancel(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	v, ok := jobs.LoadAndDelete(cmd.Process.Pid)
	if !ok {
		return
	}
	_ = windows.CloseHandle(v.(windows.Handle))
}

func cancelProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if v, ok := jobs.LoadAndDelete(cmd.Process.Pid); ok {
		job := v.(windows.Handle)
		_ = windows.TerminateJobObject(job, 1)
		_ = windows.CloseHandle(job)
		return
	}
	// A process already inside the runner's job cannot join another one.
	// Terminate the children first so a shell's sleep does not keep the pipes open.
	killTree(uint32(cmd.Process.Pid))
	_ = cmd.Process.Kill()
}

func killTree(pid uint32) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return
	}
	var children []uint32
	for {
		if entry.ParentProcessID == pid && entry.ProcessID != pid {
			children = append(children, entry.ProcessID)
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	for _, child := range children {
		killTree(child)
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_ = windows.TerminateProcess(handle, 1)
}
