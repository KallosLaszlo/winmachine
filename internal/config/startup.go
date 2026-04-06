package config

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const registryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const registryValueName = "WinMachine"

func SetAutoStart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		return k.SetStringValue(registryValueName, exePath)
	}

	err = k.DeleteValue(registryValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func IsAutoStartEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()

	_, _, err = k.GetStringValue(registryValueName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
