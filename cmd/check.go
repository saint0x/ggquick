package main

import (
	"fmt"
	"net/http"
)

func handleCheck() error {
	resp, err := http.Get("https://ggquick.fly.dev/health")
	if err != nil {
		return fmt.Errorf("server is not running: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	fmt.Println("Server is running!")
	return nil
}
