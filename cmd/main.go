package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  ggquick start              - Start the local ggquick server")
		fmt.Println("  ggquick apply [repo-url]   - Apply ggquick to a repository")
		fmt.Println("  ggquick check              - Check if ggquick server is running")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "start":
		// Start local server
		if err := handleServe(); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}

	case "apply":
		if len(os.Args) != 3 {
			fmt.Println("Usage: ggquick apply [repository-url]")
			os.Exit(1)
		}
		// Only send configuration to server
		if err := handleStart(os.Args[2]); err != nil {
			fmt.Printf("Error applying config: %v\n", err)
			os.Exit(1)
		}

	case "check":
		err = handleCheck()

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
