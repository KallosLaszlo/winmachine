package main

import (
	"embed"
	"os"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	createMutexW = kernel32.NewProc("CreateMutexW")
)

const ERROR_ALREADY_EXISTS = 183

// ensureSingleInstance creates a named mutex to prevent multiple instances
func ensureSingleInstance() (syscall.Handle, bool) {
	name, _ := syscall.UTF16PtrFromString("Global\\WinMachineBackupApp")
	handle, _, err := createMutexW.Call(0, 1, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return 0, false
	}
	// err contains GetLastError() result from the syscall
	if err.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		syscall.CloseHandle(syscall.Handle(handle))
		return 0, false
	}
	return syscall.Handle(handle), true
}

func main() {
	mutexHandle, ok := ensureSingleInstance()
	if !ok {
		// Another instance is already running
		os.Exit(0)
	}
	defer syscall.CloseHandle(mutexHandle)

	app := NewApp()
	app.trayIcon = trayIcon

	err := wails.Run(&options.App{
		Title:     "WinMachine",
		Width:     1100,
		Height:    700,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 24, B: 32, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
