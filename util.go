package main

import (
	"fmt"
	"os"
)

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
