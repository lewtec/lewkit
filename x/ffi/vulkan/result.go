package vulkan

import (
	"errors"
	"strconv"
)

var (
	// ErrUnavailable means libvulkan did not load or a required command is missing.
	ErrUnavailable = errors.New("vulkan unavailable")
	// ErrNoDevice means no compute-capable physical device was found.
	ErrNoDevice = errors.New("no compute device")
	// ErrClosed means the device, buffer, or shader was already closed.
	ErrClosed = errors.New("vulkan closed")
	// ErrSize means a buffer or write length is invalid.
	ErrSize = errors.New("invalid size")
	// ErrShader means SPIR-V or binding count is invalid.
	ErrShader = errors.New("invalid shader")
	// ErrBusy means a command buffer is recording or still on the queue.
	ErrBusy = errors.New("command buffer busy")
	// ErrPush means push-constant data does not match the shader.
	ErrPush = errors.New("invalid push constants")
)

// Result is a VkResult.
type Result int32

func (r Result) Error() string {
	if s, ok := resultName[r]; ok {
		return s
	}
	return "vulkan result " + strconv.FormatInt(int64(r), 10)
}

var resultName = map[Result]string{
	0:           "vk success",
	1:           "vk not ready",
	2:           "vk timeout",
	-1:          "vk error out of host memory",
	-2:          "vk error out of device memory",
	-3:          "vk error initialization failed",
	-4:          "vk error device lost",
	-5:          "vk error memory map failed",
	-6:          "vk error layer not present",
	-7:          "vk error extension not present",
	-8:          "vk error feature not present",
	-9:          "vk error incompatible driver",
	-10:         "vk error too many objects",
	-11:         "vk error format not supported",
	-12:         "vk error fragmented pool",
	-13:         "vk error unknown",
	-1000069000: "vk error out of pool memory",
}
