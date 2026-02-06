package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("SignalBlocks - Real-time Industrial Analytics Accelerator")
	fmt.Println("Version: 0.1.0-pre (MVP in progress)")
	fmt.Println("Founder: Masoud Rajaeei")
	fmt.Printf("Working directory: %s\n", getCurrentDir())

	// بعداً اینجا ingestion, block builder و ... اضافه می‌شه
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}