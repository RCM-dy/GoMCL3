package messageboxes

/*
#include <stdio.h>
#include <stdlib.h>
#include <windows.h>
*/
import "C"
import (
	"syscall"
	"unsafe"
)

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	messageBox = user32.NewProc("MessageBoxW")
)

func MessageBox(title, msg string, flags int) (int, error) {
	titleUTF16, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return -1, err
	}
	msgUTF16, err := syscall.UTF16PtrFromString(msg)
	if err != nil {
		return -1, err
	}
	ret, _, err := messageBox.Call(
		uintptr(C.NULL),
		uintptr(unsafe.Pointer(msgUTF16)),
		uintptr(unsafe.Pointer(titleUTF16)),
		uintptr(flags),
	)
	return int(ret), err
}
