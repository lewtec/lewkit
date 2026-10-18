//go:build windows

package exec

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// tracked holds a Job Object per started command so cancel kills the
// shell and the children that would otherwise keep its pipes open.
// The map key is the command. Wait releases the pid before finishCancel runs.
var tracked sync.Map // *exec.Cmd → *processTrack

type processTrack struct {
	job     windows.Handle
	pid     uint32
	created windows.Filetime
}

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
	rec := &processTrack{pid: uint32(cmd.Process.Pid)}
	proc, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		rec.pid,
	)
	if err != nil {
		tracked.Store(cmd, rec)
		return
	}
	defer windows.CloseHandle(proc)
	var exitTime, kernel, user windows.Filetime
	_ = windows.GetProcessTimes(proc, &rec.created, &exitTime, &kernel, &user)
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		tracked.Store(cmd, rec)
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
		tracked.Store(cmd, rec)
		return
	}
	err = windows.AssignProcessToJobObject(job, proc)
	if err != nil {
		_ = windows.CloseHandle(job)
		tracked.Store(cmd, rec)
		return
	}
	rec.job = job
	tracked.Store(cmd, rec)
}

func finishCancel(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	v, ok := tracked.LoadAndDelete(cmd)
	if !ok {
		return
	}
	rec := v.(*processTrack)
	if rec.job != 0 {
		_ = windows.CloseHandle(rec.job)
	}
}

func cancelProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if v, ok := tracked.Load(cmd); ok {
		rec := v.(*processTrack)
		if rec.job != 0 {
			_ = windows.TerminateJobObject(rec.job, 1)
			return
		}
		// TerminateProcess reports exit status 1 and prints nothing. The
		// creation time recorded at start keeps a recycled pid alive.
		if sameProcess(rec.pid, rec.created) {
			killTree(rec.pid)
		}
	}
	// Kill uses the process handle from Start, which stays with this process.
	_ = cmd.Process.Kill()
}

func sameProcess(pid uint32, created windows.Filetime) bool {
	if created.LowDateTime == 0 && created.HighDateTime == 0 {
		return false
	}
	proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(proc)
	var got, exitTime, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(proc, &got, &exitTime, &kernel, &user); err != nil {
		return false
	}
	return got.LowDateTime == created.LowDateTime && got.HighDateTime == created.HighDateTime
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
