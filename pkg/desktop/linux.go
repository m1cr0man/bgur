package desktop

import (
	"os/exec"
)

func SetBackgroundLinux(filePath string) error {
	cmd := "/usr/bin/gsettings"
	err := exec.Command(cmd, "set", "org.gnome.desktop.background", "picture-uri", "file://"+filePath).Run()
	if err != nil {
		return err
	}
	// return exec.Command(cmd, "set", "org.gnome.desktop.background", "picture-options", "zoom").Run()
	return nil
}
