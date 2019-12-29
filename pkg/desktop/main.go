package desktop

import (
	"fmt"
	"runtime"
)

func SetBackground(filePath string) error {
	switch os := runtime.GOOS; os {
	case "linux":
		return SetBackgroundLinux(filePath)
	// case "darwin":
	// case "windows":
	default:
		return fmt.Errorf("OS %s not supported. Please create a ticket on GitHub if you would like support", os)
	}
}
