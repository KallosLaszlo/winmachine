package tray

import (
	"os/exec"
	"runtime"
	"syscall"

	"github.com/energye/systray"
)

type Callbacks struct {
	OnOpenWindow  func()
	OnBackupNow   func()
	OnTogglePause func() bool // returns new paused state
	OnQuit        func()
}

var iconData []byte

func SetIcon(data []byte) {
	iconData = data
}

func Run(cb Callbacks) {
	// Lock this goroutine to its OS thread so the Windows message loop
	// (GetMessage/DispatchMessage) stays on the thread that created the
	// notification-area window.  Without this, Go's scheduler may move
	// the goroutine to another thread, which silently breaks
	// TrackPopupMenu (right-click menu) — especially on auto-start.
	runtime.LockOSThread()
	systray.Run(func() {
		onReady(cb)
	}, func() {})
}

func openURL(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}

func onReady(cb Callbacks) {
	if iconData != nil {
		systray.SetIcon(iconData)
	}
	systray.SetTitle("WinMachine")
	systray.SetTooltip("WinMachine — Time Machine for Windows")

	// Double-click tray icon to open window
	systray.SetOnDClick(func(menu systray.IMenu) {
		if cb.OnOpenWindow != nil {
			cb.OnOpenWindow()
		}
	})

	// Right-click shows menu
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	mOpen := systray.AddMenuItem("Open WinMachine", "Open the main window")
	systray.AddSeparator()
	mBackup := systray.AddMenuItem("Back Up Now", "Start a backup immediately")
	mPause := systray.AddMenuItem("Pause Backups", "Pause automatic backups")
	systray.AddSeparator()
	mAbout := systray.AddMenuItem("About", "About WinMachine")
	mWebsite := mAbout.AddSubMenuItem("Website — kallos.dev", "Open kallos.dev")
	mGitHub := mAbout.AddSubMenuItem("GitHub", "Open project on GitHub")
	mKofi := mAbout.AddSubMenuItem("Support (Ko-fi)", "Support on Ko-fi")
	mQuit := systray.AddMenuItem("Quit", "Quit WinMachine")

	mOpen.Click(func() {
		if cb.OnOpenWindow != nil {
			cb.OnOpenWindow()
		}
	})

	mBackup.Click(func() {
		if cb.OnBackupNow != nil {
			cb.OnBackupNow()
		}
	})

	mPause.Click(func() {
		if cb.OnTogglePause != nil {
			paused := cb.OnTogglePause()
			if paused {
				mPause.SetTitle("Resume Backups")
			} else {
				mPause.SetTitle("Pause Backups")
			}
		}
	})

	mWebsite.Click(func() {
		openURL("https://kallos.dev")
	})
	mGitHub.Click(func() {
		openURL("https://github.com/KallosLaszlo/winmachine")
	})
	mKofi.Click(func() {
		openURL("https://ko-fi.com/laszlokallos")
	})

	mQuit.Click(func() {
		if cb.OnQuit != nil {
			cb.OnQuit()
		}
		systray.Quit()
	})
}
