//go:build darwin

package main

import (
	"fmt"
	"strings"
	"time"

	"mux/internal/accounts"
)

func (a *App) updateStatusBarUsage() {
	if a.store == nil {
		return
	}
	var total, used float64
	var details strings.Builder
	for _, account := range a.store.List() {
		var window *accounts.UsageWindow
		if account.Usage != nil {
			window = account.Usage.PrimaryWindow
		}
		if window == nil {
			fmt.Fprintf(&details, "%s：暂无用量数据\n", account.Name)
			continue
		}
		total++
		used += window.UsedPercent
		fmt.Fprintf(&details, "%s：剩余 %.0f%%，%s 重置\n", account.Name, 100-window.UsedPercent, time.Unix(window.ResetAt, 0).Format("01/02 15:04"))
	}
	setStatusBarDetails(strings.TrimSuffix(details.String(), "\n"))
	if total == 0 {
		setStatusBarTitle("Mux")
		return
	}
	setStatusBarTitle(fmt.Sprintf("Mux %.0f%%", used/total))
}
