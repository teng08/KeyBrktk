//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	registry       = syscall.NewLazyDLL("advapi32.dll")
	openRegistry   = registry.NewProc("RegOpenKeyExW")
	createRegistry = registry.NewProc("RegCreateKeyExW")
	queryRegistry  = registry.NewProc("RegQueryValueExW")
	setRegistry    = registry.NewProc("RegSetValueExW")
	deleteRegistry = registry.NewProc("RegDeleteValueW")
	closeRegistry  = registry.NewProc("RegCloseKey")
)

const startupKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func loginCommand() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return `"` + path + `" --background`
}
func loginEnabled() bool {
	var key uintptr
	result, _, _ := openRegistry.Call(^uintptr(0x7ffffffe), uintptr(unsafe.Pointer(wide(startupKey))), 0, 0x20019, uintptr(unsafe.Pointer(&key)))
	if result != 0 {
		return false
	}
	defer closeRegistry.Call(key)
	var data [32768]uint16
	size := uint32(len(data) * 2)
	var kind uint32
	result, _, _ = queryRegistry.Call(key, uintptr(unsafe.Pointer(wide("Keybed"))), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)))
	return result == 0 && kind == 1 && syscall.UTF16ToString(data[:]) == loginCommand()
}
func setLoginEnabled(enabled bool) error {
	var key uintptr
	result, _, _ := createRegistry.Call(^uintptr(0x7ffffffe), uintptr(unsafe.Pointer(wide(startupKey))), 0, 0, 0, 0x20006, 0, uintptr(unsafe.Pointer(&key)), 0)
	if result != 0 {
		return fmt.Errorf("open login setting: %w", syscall.Errno(result))
	}
	defer closeRegistry.Call(key)
	if enabled {
		command := loginCommand()
		if command == "" {
			return fmt.Errorf("cannot locate Keybed.exe")
		}
		data, err := syscall.UTF16FromString(command)
		if err != nil {
			return err
		}
		result, _, _ = setRegistry.Call(key, uintptr(unsafe.Pointer(wide("Keybed"))), 0, 1, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	} else {
		result, _, _ = deleteRegistry.Call(key, uintptr(unsafe.Pointer(wide("Keybed"))))
		if result == 2 {
			return nil
		}
	}
	if result != 0 {
		return fmt.Errorf("update login setting: %w", syscall.Errno(result))
	}
	return nil
}
