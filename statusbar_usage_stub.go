//go:build !darwin

package main

func startStatusBar()                {}
func stopStatusBar()                 {}
func (a *App) updateStatusBarUsage() {}
