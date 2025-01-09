package main

import (
	"fmt"
	"time"
)

// Example demonstrates various Go features
func Example() {
	// Basic variables
	name := "World"
	count := 42

	// String formatting
	fmt.Printf("Hello, %s!\n", name)
	fmt.Printf("The answer is %d\n", count)

	// Time operations
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	fmt.Printf("Tomorrow will be %s\n", tomorrow.Format("2006-01-02"))

	// Slice operations
	numbers := []int{1, 2, 3, 4, 5}
	doubled := make([]int, len(numbers))
	for i, n := range numbers {
		doubled[i] = n * 2
	}
	fmt.Printf("Doubled numbers: %v\n", doubled)

	// Map operations
	scores := map[string]int{
		"Alice": 95,
		"Bob":   87,
		"Carol": 92,
	}
	for name, score := range scores {
		fmt.Printf("%s scored %d\n", name, score)
	}
}

func main() {
	Example()
} 