package bundled

import "fmt"

func AppPathsForUI(appName string) (string, error) {
	dir, err := appBinDir(appName)
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("empty bin dir")
	}
	return dir, nil
}
