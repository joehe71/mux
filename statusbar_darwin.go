//go:build darwin

package main

/*
#include <stdlib.h>
#cgo darwin LDFLAGS: -framework Cocoa
void muxStatusBarStart(void);
void muxStatusBarSetTitle(const char *title);
void muxStatusBarSetUsage(double usedPercent);
void muxStatusBarSetDetails(const char *details);
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

func setStatusBarUsage(usedPercent float64) {
	C.muxStatusBarSetUsage(C.double(usedPercent))
}

func setStatusBarDetails(details string) {
	cDetails := C.CString(details)
	defer C.free(unsafe.Pointer(cDetails))
	C.muxStatusBarSetDetails(cDetails)
}

func stopStatusBar() {
	C.muxStatusBarStop()
}
