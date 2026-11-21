package ebitenbackend

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32                 = windows.NewLazySystemDLL("gdi32.dll")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procCreateDCW         = gdi32.NewProc("CreateDCW")
	procGetDeviceCaps     = gdi32.NewProc("GetDeviceCaps")
	procDeleteDC          = gdi32.NewProc("DeleteDC")
)

const (
	monitorDefaultToNearest = 2
	vRefresh                = 116 // GetDeviceCaps index of the vertical refresh rate
)

// monitorInfoEx is MONITORINFOEXW.
type monitorInfoEx struct {
	cbSize    uint32
	rcMonitor [4]int32
	rcWork    [4]int32
	dwFlags   uint32
	szDevice  [32]uint16
}

// refreshRate is the refresh rate of the monitor this process's window is
// mostly on, or 0 when it cannot be found. Ebitengine knows it but does not
// export it, so it is asked of the monitor's display device directly: these
// are plain Win32 calls, safe from any thread and not routed through
// Ebitengine's main thread.
func refreshRate() int {
	h := ownWindow()
	if h == 0 {
		return 0
	}
	for _, p := range []*windows.LazyProc{procMonitorFromWindow, procGetMonitorInfoW, procCreateDCW, procGetDeviceCaps, procDeleteDC} {
		if p.Find() != nil {
			return 0
		}
	}
	mon, _, _ := procMonitorFromWindow.Call(uintptr(h), monitorDefaultToNearest)
	if mon == 0 {
		return 0
	}
	var mi monitorInfoEx
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if ok, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return 0
	}
	dev := &mi.szDevice[0]
	dc, _, _ := procCreateDCW.Call(uintptr(unsafe.Pointer(dev)), uintptr(unsafe.Pointer(dev)), 0, 0)
	if dc == 0 {
		return 0
	}
	defer procDeleteDC.Call(dc)
	hz, _, _ := procGetDeviceCaps.Call(dc, vRefresh)
	// 0 and 1 mean the hardware's default rate, which is not a number.
	if int32(hz) <= 1 {
		return 0
	}
	return int(int32(hz))
}
