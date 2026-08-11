//go:build windows

package main

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	settingsBaseWidth       = 640
	settingsBaseHeight      = 420
	settingsBasePadding     = 24
	settingsBaseTitleHeight = 48
)

type settingsLayout struct {
	ClientWidth  int
	ClientHeight int
	Padding      int
	TitleHeight  int
	Resizable    bool
	Maximizable  bool
}

type settingsModel struct {
	Title       string
	Description string
	Status      string
}

func scaleDPI(value, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return (value*dpi + 48) / 96
}

func defaultSettingsLayout(dpi int) settingsLayout {
	return settingsLayout{
		ClientWidth:  scaleDPI(settingsBaseWidth, dpi),
		ClientHeight: scaleDPI(settingsBaseHeight, dpi),
		Padding:      scaleDPI(settingsBasePadding, dpi),
		TitleHeight:  scaleDPI(settingsBaseTitleHeight, dpi),
		Resizable:    false,
		Maximizable:  false,
	}
}

func centeredPosition(screenWidth, screenHeight, windowWidth, windowHeight int) (int, int) {
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func defaultSettingsModel() settingsModel {
	return settingsModel{
		Title:       PluginName + " 设置",
		Description: "这是插件设置窗口模板，可在 settings.go 中修改说明和业务配置。",
		Status:      "插件运行正常",
	}
}

const (
	synchronize       = 0x00100000
	wmDestroy         = 0x0002
	wmClose           = 0x0010
	wmPaint           = 0x000f
	wmEraseBkgnd      = 0x0014
	wmGetMinMaxInfo   = 0x0024
	swShow            = 5
	swRestore         = 9
	wsExTopmost       = 0x00000008
	wsOverlapped      = 0x00000000
	wsCaption         = 0x00c00000
	wsSysMenu         = 0x00080000
	wsMinimizeBox     = 0x00020000
	cwUseDefault      = 0x80000000
	colorWindow       = 5
	idiApplication    = 32512
	idcArrow          = 32512
	dtLeft            = 0x00000000
	dtCenter          = 0x00000001
	dtVCenter         = 0x00000004
	dtSingleLine      = 0x00000020
	dtWordBreak       = 0x00000010
	dtNoPrefix        = 0x00000800
	transparent       = 1
	fwNormal          = 400
	fwSemiBold        = 600
	logPixelsX        = 88
	smCxScreen        = 0
	mbOK              = 0
	mbIconInformation = 0x40
	swpNoSize         = 0x0001
	swpNoMove         = 0x0002
	swpShowWindow     = 0x0040
)

var hwndTopmost = ^uintptr(0)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	user32                  = syscall.NewLazyDLL("user32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	gdiplus                 = syscall.NewLazyDLL("gdiplus.dll")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procUnregisterClassW    = user32.NewProc("UnregisterClassW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetWindowPlacement  = user32.NewProc("GetWindowPlacement")
	procIsIconic            = user32.NewProc("IsIconic")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procAdjustWindowRectEx  = user32.NewProc("AdjustWindowRectEx")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procDrawTextW           = user32.NewProc("DrawTextW")
	procFillRect            = user32.NewProc("FillRect")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procGetDeviceCaps       = gdi32.NewProc("GetDeviceCaps")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	openProcess             = kernel32.NewProc("OpenProcess")
	waitSingleObject        = kernel32.NewProc("WaitForSingleObject")
	closeHandle             = kernel32.NewProc("CloseHandle")
	procCreateFontW         = gdi32.NewProc("CreateFontW")
	procSelectObject        = gdi32.NewProc("SelectObject")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procGdiplusStartup      = gdiplus.NewProc("GdiplusStartup")
	procGdiplusShutdown     = gdiplus.NewProc("GdiplusShutdown")
	procGdipCreateFromHDC   = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics  = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothing    = gdiplus.NewProc("GdipSetSmoothingMode")
	procGdipCreateSolidFill = gdiplus.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush     = gdiplus.NewProc("GdipDeleteBrush")
	procGdipFillRectangleI  = gdiplus.NewProc("GdipFillRectangleI")
	procGdipFillEllipseI    = gdiplus.NewProc("GdipFillEllipseI")
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type windowPlacement struct {
	Length, Flags, ShowCmd   uint32
	MinPosition, MaxPosition point
	NormalPosition           rect
}
type msg struct {
	HWnd, Message, WParam, LParam uintptr
	Time                          uint32
	Pt                            point
	Private                       uint32
}
type paintStruct struct {
	HDC, Erase, Restore, IncUpdate uintptr
	Paint                          rect
	Reserved                       [32]byte
}
type wndClassEx struct {
	Size, Style          uint32
	WndProc, ClsExtra    uintptr
	WndExtra, Instance   uintptr
	Icon, Cursor         uintptr
	Background, MenuName uintptr
	ClassName, IconSmall uintptr
}
type gdiplusStartupInput struct {
	Version                  uint32
	Debug                    uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}
type minMaxInfo struct {
	Reserved, MaxSize, MaxPosition, MinTrackSize, MaxTrackSize point
}

var settingsNative = struct {
	sync.Mutex
	hwnd    uintptr
	done    chan struct{}
	closing bool
}{}

var settingsWndProc = syscall.NewCallback(settingsWindowProc)

func showSettingsWindow() {
	settingsNative.Lock()
	if settingsNative.hwnd != 0 {
		hwnd := settingsNative.hwnd
		settingsNative.Unlock()
		iconic, _, _ := procIsIconic.Call(hwnd)
		if iconic != 0 {
			procShowWindow.Call(hwnd, swRestore)
		}
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
		procSetForegroundWindow.Call(hwnd)
		return
	}
	if settingsNative.done != nil || settingsNative.closing {
		settingsNative.Unlock()
		return
	}
	settingsNative.done = make(chan struct{})
	done := settingsNative.done
	settingsNative.Unlock()
	go settingsWindowThread(done)
}

func closeSettingsWindow() {
	settingsNative.Lock()
	done, hwnd := settingsNative.done, settingsNative.hwnd
	if done == nil {
		settingsNative.Unlock()
		return
	}
	settingsNative.closing = true
	settingsNative.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
	<-done
}

func settingsWindowThread(done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func() {
		settingsNative.Lock()
		settingsNative.hwnd = 0
		settingsNative.done = nil
		settingsNative.closing = false
		close(done)
		settingsNative.Unlock()
	}()

	instance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString("BeeGoSettingsWindow")
	title, _ := syscall.UTF16PtrFromString(defaultSettingsModel().Title)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	icon, _, _ := procLoadIconW.Call(0, idiApplication)
	class := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: settingsWndProc, Instance: instance, Icon: icon, Cursor: cursor, Background: colorWindow + 1, ClassName: uintptr(unsafe.Pointer(className)), IconSmall: icon}
	atom, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		return
	}
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), instance)

	dpi := settingsDPI()
	layout := defaultSettingsLayout(dpi)
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	windowRect := rect{Right: int32(layout.ClientWidth), Bottom: int32(layout.ClientHeight)}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&windowRect)), style, 0, 0)
	width, height := int(windowRect.Right-windowRect.Left), int(windowRect.Bottom-windowRect.Top)
	screenW, _, _ := procGetSystemMetrics.Call(smCxScreen)
	screenH, _, _ := procGetSystemMetrics.Call(1)
	x, y := centeredPosition(int(screenW), int(screenH), width, height)
	hwnd, _, _ := procCreateWindowExW.Call(wsExTopmost, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), style, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, instance, 0)
	if hwnd == 0 {
		return
	}
	settingsNative.Lock()
	settingsNative.hwnd = hwnd
	settingsNative.Unlock()
	procShowWindow.Call(hwnd, swShow)
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	procSetForegroundWindow.Call(hwnd)
	procUpdateWindow.Call(hwnd)
	var message msg
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func settingsDPI() int {
	dc, _, _ := procGetDC.Call(0)
	if dc == 0 {
		return 96
	}
	defer procReleaseDC.Call(0, dc)
	dpi, _, _ := procGetDeviceCaps.Call(dc, logPixelsX)
	if dpi == 0 {
		return 96
	}
	return int(dpi)
}

func settingsWindowProc(hwnd uintptr, message uint32, wparam, lparam uintptr) uintptr {
	switch message {
	case wmEraseBkgnd:
		return 1
	case wmGetMinMaxInfo:
		info := *(**minMaxInfo)(unsafe.Pointer(&lparam))
		var placement windowPlacement
		placement.Length = uint32(unsafe.Sizeof(placement))
		procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
		width := placement.NormalPosition.Right - placement.NormalPosition.Left
		height := placement.NormalPosition.Bottom - placement.NormalPosition.Top
		if width > 0 && height > 0 {
			info.MinTrackSize = point{width, height}
			info.MaxTrackSize = point{width, height}
		}
		return 0
	case wmPaint:
		paintSettings(hwnd)
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wparam, lparam)
	return result
}

func paintSettings(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	dpi := settingsDPI()
	layout := defaultSettingsLayout(dpi)
	model := defaultSettingsModel()
	background, _, _ := procCreateSolidBrush.Call(rgb(246, 247, 251))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), background)
	procDeleteObject.Call(background)

	var token, graphics, whiteBrush, accentBrush, successBrush uintptr
	input := gdiplusStartupInput{Version: 1}
	status, _, _ := procGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&input)), 0)
	if status == 0 {
		procGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&graphics)))
		procGdipSetSmoothing.Call(graphics, 4)
		procGdipCreateSolidFill.Call(argb(255, 255, 255, 255), uintptr(unsafe.Pointer(&whiteBrush)))
		procGdipCreateSolidFill.Call(argb(255, 99, 102, 241), uintptr(unsafe.Pointer(&accentBrush)))
		procGdipCreateSolidFill.Call(argb(255, 18, 183, 106), uintptr(unsafe.Pointer(&successBrush)))
		p := layout.Padding
		procGdipFillRectangleI.Call(graphics, whiteBrush, uintptr(p), uintptr(layout.TitleHeight+p), uintptr(layout.ClientWidth-2*p), uintptr(layout.ClientHeight-layout.TitleHeight-2*p))
		procGdipFillRectangleI.Call(graphics, accentBrush, uintptr(p), uintptr(p), uintptr(scaleDPI(6, dpi)), uintptr(layout.TitleHeight-scaleDPI(8, dpi)))
		procGdipFillEllipseI.Call(graphics, successBrush, uintptr(p+scaleDPI(22, dpi)), uintptr(layout.TitleHeight+p+scaleDPI(42, dpi)), uintptr(scaleDPI(12, dpi)), uintptr(scaleDPI(12, dpi)))
		procGdipDeleteBrush.Call(successBrush)
		procGdipDeleteBrush.Call(accentBrush)
		procGdipDeleteBrush.Call(whiteBrush)
		procGdipDeleteGraphics.Call(graphics)
		procGdiplusShutdown.Call(token)
	}

	fontName, _ := syscall.UTF16PtrFromString("Microsoft YaHei UI")
	titleFont, _, _ := procCreateFontW.Call(uintptr(-scaleDPI(22, dpi)), 0, 0, 0, fwSemiBold, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(fontName)))
	bodyFont, _, _ := procCreateFontW.Call(uintptr(-scaleDPI(14, dpi)), 0, 0, 0, fwNormal, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(fontName)))
	procSetBkMode.Call(hdc, transparent)
	drawWindowText(hdc, model.Title, rect{int32(layout.Padding + scaleDPI(22, dpi)), int32(layout.Padding), int32(layout.ClientWidth - layout.Padding), int32(layout.Padding + layout.TitleHeight)}, titleFont, rgb(24, 34, 48), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawWindowText(hdc, model.Status, rect{int32(layout.Padding + scaleDPI(42, dpi)), int32(layout.TitleHeight + layout.Padding + scaleDPI(30, dpi)), int32(layout.ClientWidth - layout.Padding), int32(layout.TitleHeight + layout.Padding + scaleDPI(68, dpi))}, bodyFont, rgb(18, 122, 77), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawWindowText(hdc, model.Description, rect{int32(layout.Padding + scaleDPI(22, dpi)), int32(layout.TitleHeight + layout.Padding + scaleDPI(92, dpi)), int32(layout.ClientWidth - layout.Padding - scaleDPI(22, dpi)), int32(layout.ClientHeight - layout.Padding)}, bodyFont, rgb(102, 112, 133), dtLeft|dtWordBreak|dtNoPrefix)
	if titleFont != 0 {
		procDeleteObject.Call(titleFont)
	}
	if bodyFont != 0 {
		procDeleteObject.Call(bodyFont)
	}
}

func drawWindowText(hdc uintptr, text string, area rect, font uintptr, color uintptr, flags uintptr) {
	value, _ := syscall.UTF16PtrFromString(text)
	old, _, _ := procSelectObject.Call(hdc, font)
	procSetTextColor.Call(hdc, color)
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(value)), ^uintptr(0), uintptr(unsafe.Pointer(&area)), flags)
	procSelectObject.Call(hdc, old)
}

func rgb(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }
func argb(a, r, g, b byte) uintptr {
	return uintptr(uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func startHostWatcher(hostPID uint32) {
	handle, _, _ := openProcess.Call(synchronize, 0, uintptr(hostPID))
	if handle == 0 {
		return
	}
	go func() {
		defer closeHandle.Call(handle)
		waitSingleObject.Call(handle, 0xffffffff)
		os.Exit(0)
	}()
}
