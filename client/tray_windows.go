//go:build windows && amd64

package main

import (
	"context"
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	trayCallbackMessage = 0x0400 + 1
	trayOpenCommand     = 1001
	trayExitCommand     = 1002
	wmTrayNotify        = trayCallbackMessage
	wmLButtonUp         = 0x0202
	wmRButtonUp         = 0x0205
	wmCommand           = 0x0111
	wmClose             = 0x0010
	wmQuit              = 0x0012
	nimAdd              = 0x00000000
	nimDelete           = 0x00000002
	nifMessage          = 0x00000001
	nifIcon             = 0x00000002
	nifTip              = 0x00000004
	tpmRightButton      = 0x0002
	idiApplication      = 32512
	hwndMessage         = ^uintptr(2)
)

type trayPoint struct {
	X int32
	Y int32
}

type trayMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   trayPoint
}

type trayWndClass struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	SmallIcon  uintptr
}

type trayIconData struct {
	Size           uint32
	Wnd            uintptr
	ID             uint32
	Flags          uint32
	Callback       uint32
	Icon           uintptr
	Tip            [128]uint16
	State          uint32
	StateMask      uint32
	Info           [256]uint16
	Timeout        uint32
	InfoFlags      uint32
	Guid           [16]byte
	BalloonTitle   [64]uint16
	BalloonInfo    [256]uint16
	BalloonTimeout uint32
	Reserved       uint32
	Reserved2      uint32
}

type systemTray struct {
	mu       sync.Mutex
	ctx      context.Context
	hwnd     uintptr
	callback uintptr
	closed   bool
}

func newSystemTray() *systemTray { return &systemTray{} }

func (t *systemTray) Start(ctx context.Context) {
	t.mu.Lock()
	t.ctx = ctx
	t.mu.Unlock()
	go t.run()
}

func (t *systemTray) Close() {
	t.mu.Lock()
	t.closed = true
	hwnd := t.hwnd
	t.mu.Unlock()
	if hwnd != 0 {
		postMessage.Call(hwnd, wmClose, 0, 0)
	}
}

func (t *systemTray) run() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	registerClass := user32.NewProc("RegisterClassExW")
	createWindow := user32.NewProc("CreateWindowExW")
	defWindowProc := user32.NewProc("DefWindowProcW")
	getMessage := user32.NewProc("GetMessageW")
	translateMessage := user32.NewProc("TranslateMessage")
	dispatchMessage := user32.NewProc("DispatchMessageW")
	destroyWindow := user32.NewProc("DestroyWindow")
	postQuit := user32.NewProc("PostQuitMessage")
	loadIcon := user32.NewProc("LoadIconW")
	shellNotify := shell32.NewProc("Shell_NotifyIconW")
	createMenu := user32.NewProc("CreatePopupMenu")
	appendMenu := user32.NewProc("AppendMenuW")
	trackMenu := user32.NewProc("TrackPopupMenu")
	destroyMenu := user32.NewProc("DestroyMenu")
	getCursor := user32.NewProc("GetCursorPos")

	className, _ := windows.UTF16PtrFromString("SogameNetBirdTrayWindow")
	t.callback = windows.NewCallback(func(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
		switch message {
		case wmTrayNotify:
			switch lParam {
			case wmLButtonUp:
				t.showWindow()
			case wmRButtonUp:
				point := trayPoint{}
				getCursor.Call(uintptr(unsafe.Pointer(&point)))
				menu, _, _ := createMenu.Call()
				if menu != 0 {
					openText, _ := windows.UTF16PtrFromString("打开 Sogame")
					exitText, _ := windows.UTF16PtrFromString("退出 Sogame")
					appendMenu.Call(menu, 0, trayOpenCommand, uintptr(unsafe.Pointer(openText)))
					appendMenu.Call(menu, 0, trayExitCommand, uintptr(unsafe.Pointer(exitText)))
					trackMenu.Call(menu, tpmRightButton, uintptr(point.X), uintptr(point.Y), 0, hwnd, 0)
					destroyMenu.Call(menu)
				}
			}
		case wmCommand:
			switch wParam & 0xffff {
			case trayOpenCommand:
				t.showWindow()
			case trayExitCommand:
				t.quitApp()
			}
		case wmClose:
			if shellNotify != nil {
				// The icon is removed below after the message loop exits.
			}
			destroyWindow.Call(hwnd)
			postQuit.Call(0)
		default:
			result, _, _ := defWindowProc.Call(hwnd, uintptr(message), wParam, lParam)
			return result
		}
		return 0
	})

	instance, _, _ := getModuleHandle.Call(0)
	class := trayWndClass{Size: uint32(unsafe.Sizeof(trayWndClass{})), WndProc: t.callback, Instance: instance, ClassName: className}
	registerClass.Call(uintptr(unsafe.Pointer(&class)))
	hwnd, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, hwndMessage, 0, instance, 0)
	if hwnd == 0 {
		return
	}
	t.mu.Lock()
	t.hwnd = hwnd
	t.mu.Unlock()
	icon, _, _ := loadIcon.Call(0, idiApplication)
	tip, _ := windows.UTF16FromString("Sogame NetBird")
	data := trayIconData{Size: uint32(unsafe.Sizeof(trayIconData{})), Wnd: hwnd, ID: 1, Flags: nifMessage | nifIcon | nifTip, Callback: wmTrayNotify, Icon: icon}
	copy(data.Tip[:], tip)
	shellNotify.Call(nimAdd, uintptr(unsafe.Pointer(&data)))

	message := trayMsg{}
	for {
		messageResult, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if messageResult <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	shellNotify.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	t.mu.Lock()
	t.hwnd = 0
	t.mu.Unlock()
}

func (t *systemTray) showWindow() {
	t.mu.Lock()
	ctx := t.ctx
	t.mu.Unlock()
	if ctx != nil {
		runtime.WindowShow(ctx)
	}
}

func (t *systemTray) quitApp() {
	t.mu.Lock()
	ctx := t.ctx
	t.mu.Unlock()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

var (
	getModuleHandle = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW")
	postMessage     = windows.NewLazySystemDLL("user32.dll").NewProc("PostMessageW")
)
