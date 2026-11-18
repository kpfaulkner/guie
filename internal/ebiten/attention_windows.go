package ebitenbackend

import (
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	procFlashWindowEx = user32.NewProc("FlashWindowEx")

	hwndOnce sync.Once
	hwnd     windows.HWND
)

// glfwClass is the window class Ebitengine's GLFW port registers.
const glfwClass = "GLFW30"

const (
	flashwTray      = 0x2 // flash the taskbar button
	flashwTimerNoFG = 0xC // until the window comes to the foreground
)

type flashWInfo struct {
	cbSize    uint32
	hwnd      windows.HWND
	dwFlags   uint32
	uCount    uint32
	dwTimeout uint32
}

// ownWindow finds this process's GLFW window. Ebitengine does not hand out its
// handle, so the top-level windows are searched once for one of its class
// owned by this process.
func ownWindow() windows.HWND {
	hwndOnce.Do(func() {
		pid := uint32(os.Getpid())
		cb := windows.NewCallback(func(h windows.HWND, _ uintptr) uintptr {
			var owner uint32
			if _, err := windows.GetWindowThreadProcessId(h, &owner); err != nil || owner != pid {
				return 1
			}
			buf := make([]uint16, 64)
			n, err := windows.GetClassName(h, &buf[0], int32(len(buf)))
			if err != nil || windows.UTF16ToString(buf[:n]) != glfwClass {
				return 1
			}
			hwnd = h
			return 0 // stop
		})
		_ = windows.EnumWindows(cb, nil) // returns an error when the callback stops it
	})
	return hwnd
}

// requestAttention flashes the taskbar button until the window is brought
// forward.
func requestAttention() bool {
	h := ownWindow()
	if h == 0 || procFlashWindowEx.Find() != nil {
		return false
	}
	info := flashWInfo{hwnd: h, dwFlags: flashwTray | flashwTimerNoFG}
	info.cbSize = uint32(unsafe.Sizeof(info))
	procFlashWindowEx.Call(uintptr(unsafe.Pointer(&info)))
	return true
}
