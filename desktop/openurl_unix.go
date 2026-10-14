//go:build !windows

package main

import (
	"os/exec"
	"runtime"
	"syscall"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenURL opens the given URL in the system browser without blocking the Wails bridge.
// Uses os/exec with .Start() (detached) instead of Wails' synchronous BrowserOpenURL,
// which can block on Linux when xdg-open waits for the browser process.
func (a *App) OpenURL(url string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	// Detach so the Go process doesn't wait for the browser to finish.
	if cmd != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	// Fallback: Wails runtime (may block on xdg-open but works everywhere)
	wailsRuntime.BrowserOpenURL(a.ctx, url)
	return nil
}
