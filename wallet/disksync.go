package wallet

import (
	"fmt"
	"os"
)

func CheckCreateDir(path string) error {
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, 0700); err != nil {
				return fmt.Errorf("cannot create directory: %w", err)
			}
		} else {
			return fmt.Errorf("error checking directory: %w", err)
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}

	return nil
}
