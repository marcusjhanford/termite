package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/spf13/cobra"
)

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>land.charm.termite</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Binary}}</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/daemon.log</string>
</dict>
</plist>`

const systemdUnit = `[Unit]
Description=Termite Email Sync Daemon
After=network-online.target

[Service]
ExecStart={{.Binary}} daemon
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target`

type serviceTemplateData struct {
	Binary string
	LogDir string
}

var installDaemonCmd = &cobra.Command{
	Use:   "install-daemon",
	Short: "Install Termite as a background sync service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find executable path: %w", err)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to find home directory: %w", err)
		}

		data := serviceTemplateData{
			Binary: binary,
			LogDir: filepath.Join(home, ".termite"),
		}

		switch runtime.GOOS {
		case "darwin":
			return installLaunchd(home, data)
		case "linux":
			return installSystemd(home, data)
		default:
			return fmt.Errorf("unsupported OS: %s (only macOS and Linux are supported)", runtime.GOOS)
		}
	},
}

func installLaunchd(home string, data serviceTemplateData) error {
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "land.charm.termite.plist")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl := template.Must(template.New("plist").Parse(launchdPlist))
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Installed launchd service: %s\n", path)
	fmt.Println("Run: launchctl load", path)
	return nil
}

func installSystemd(home string, data serviceTemplateData) error {
	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "termite.service")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl := template.Must(template.New("unit").Parse(systemdUnit))
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Installed systemd user service: %s\n", path)
	fmt.Println("Run: systemctl --user enable --now termite")
	return nil
}

func init() {
	rootCmd.AddCommand(installDaemonCmd)
}
