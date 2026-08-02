//go:build darwin

package main

/*
#include <stdlib.h>
#cgo darwin LDFLAGS: -framework Cocoa
void muxStatusBarStart(void);
void muxStatusBarSetTitle(const char *title);
void muxStatusBarStop(void);
*/
import "C"

import "unsafe"

func startStatusBar() {
	C.muxStatusBarStart()
}

func setStatusBarTitle(title string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.muxStatusBarSetTitle(cTitle)
}

func stopStatusBar() {
	C.muxStatusBarStop()
}
