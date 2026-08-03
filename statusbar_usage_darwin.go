//go:build darwin

package main

import "fmt"

func (a *App) updateStatusBarUsage() {
	if a.store == nil {
		return
	}
	var total, used float64
	for _, account := range a.store.List() {
		if account.Usage == nil || account.Usage.PrimaryWindow == nil {
			continue
		}
		total++
		used += account.Usage.PrimaryWindow.UsedPercent
	}
	if total == 0 {
		setStatusBarTitle("Mux")
		return
	}
	setStatusBarTitle(fmt.Sprintf("Mux %.0f%%", used/total))
}
