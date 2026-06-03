package winjob

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Job struct {
	handle windows.Handle
}

const (
	JobObjectExtendedLimitInformation  = 9
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x2000
)

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	SchedulingClass         uint32
	PriorityClass           uint32
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

func CreateJob() (*Job, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create job object: %w", err)
	}

	var info jobObjectExtendedLimitInformation
	info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	_, err = windows.SetInformationJobObject(
		h,
		JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("failed to set job object info: %w", err)
	}

	return &Job{handle: h}, nil
}

func (j *Job) AssignProcess(pid int) error {
	hProcess, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("failed to open process %d: %w", err)
	}
	defer windows.CloseHandle(hProcess)

	err = windows.AssignProcessToJobObject(j.handle, hProcess)
	if err != nil {
		return fmt.Errorf("failed to assign process to job object: %w", err)
	}

	return nil
}

func (j *Job) Close() {
	if j.handle != 0 {
		windows.CloseHandle(j.handle)
		j.handle = 0
	}
}
