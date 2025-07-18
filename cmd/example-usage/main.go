package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define callback to receive Wiegand data
	callback := func(data string) {
		fmt.Println("Received Wiegand data:", data)
	}

	// Configure Wiegand reader (use actual Raspberry Pi GPIO pins, e.g., GPIO14 and GPIO15)
	cfg := wiegand.Config{
		D0Pin:    "GPIO4",  // Wiegand D0 (e.g., green wire)
		D1Pin:    "GPIO17", // Wiegand D1 (e.g., white wire)
		Callback: callback,
		Timeout:  100 * time.Millisecond,
		MaxBits:  26,
	}

	// Initialize reader
	reader, err := wiegand.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Wiegand reader: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Keep program running to process Wiegand data
	select {
	case <-ctx.Done():
		fmt.Println("Shutting down Wiegand reader")
	}
}
