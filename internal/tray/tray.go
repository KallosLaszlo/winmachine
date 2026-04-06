package tray

import (
	"github.com/energye/systray"
)

type Callbacks struct {
	OnOpenWindow func()
	OnBackupNow  func()
	OnTogglePause func() bool // returns new paused state
	OnQuit       func()
}

var iconData []byte

func SetIcon(data []byte) {
	iconData = data
}

func Run(cb Callbacks) {
	systray.Run(func() {
		onReady(cb)
	}, func() {})
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

	mQuit.Click(func() {
		if cb.OnQuit != nil {
			cb.OnQuit()
		}
		systray.Quit()
	})
}
