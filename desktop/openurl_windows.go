//go:build windows

package main

import (
	"os/exec"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenURL opens the given URL in the system browser without blocking the Wails bridge.
// rundll32 returns immediately, so no process-group detachment is needed
// (and syscall.SysProcAttr{Setpgid} does not exist on Windows).
func (a *App) OpenURL(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err == nil {
		return nil
	}
	// Fallback: Wails runtime
	wailsRuntime.BrowserOpenURL(a.ctx, url)
	return nil
}
