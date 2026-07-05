package notification

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Send sends a cross-platform desktop notification.
// It runs non-blockingly and ignores errors to avoid crashing or halting the app.
func Send(title, message string) {
	go func() {
		switch runtime.GOOS {
		case "linux":
			if _, err := exec.LookPath("notify-send"); err == nil {
				_ = exec.Command("notify-send", "-a", "GoYT", title, message).Run()
			}
		case "darwin":
			script := fmt.Sprintf("display notification %q with title %q", message, title)
			_ = exec.Command("osascript", "-e", script).Run()
		case "windows":
			psCmd := fmt.Sprintf(
				`[void] [System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms"); ` +
				`$objNotification = New-Object System.Windows.Forms.NotifyIcon; ` +
				`$objNotification.Icon = [System.Drawing.SystemIcons]::Information; ` +
				`$objNotification.BalloonTipIcon = "Info"; ` +
				`$objNotification.BalloonTipText = %q; ` +
				`$objNotification.BalloonTipTitle = %q; ` +
				`$objNotification.Visible = $True; ` +
				`$objNotification.ShowBalloonTip(5000)`,
				message, title,
			)
			_ = exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run()
		}
	}()
}
