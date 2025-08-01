package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asjoyner/wiegand-go"
)

func main() {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define callback to receive Wiegand data
	callback := func(site, tag string) {
		fmt.Println("Received Wiegand data: site: %s, tag: %s", site, tag)
	}

	reader1, err := wiegand.New(ctx, wiegand.Config{
		D0Pin:    "GPIO4",  // Wiegand D0 (e.g., green wire)
		D1Pin:    "GPIO17", // Wiegand D1 (e.g., white wire)
		Callback: callback,
		Timeout:  100 * time.Millisecond,
		MaxBits:  26,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Wiegand reader: %v\n", err)
		os.Exit(1)
	}
	defer reader1.Close()

	reader2, err := wiegand.New(ctx, wiegand.Config{
		D0Pin:    "GPIO18",
		D1Pin:    "GPIO27",
		Callback: callback,
		Timeout:  100 * time.Millisecond,
		MaxBits:  26,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Wiegand reader: %v\n", err)
		os.Exit(1)
	}
	defer reader2.Close()

	reader3, err := wiegand.New(ctx, wiegand.Config{
		D0Pin:    "GPIO22",
		D1Pin:    "GPIO23",
		Callback: callback,
		Timeout:  100 * time.Millisecond,
		MaxBits:  26,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Wiegand reader: %v\n", err)
		os.Exit(1)
	}
	defer reader3.Close()

	reader4, err := wiegand.New(ctx, wiegand.Config{
		D0Pin:    "GPIO24",
		D1Pin:    "GPIO25",
		Callback: callback,
		Timeout:  100 * time.Millisecond,
		MaxBits:  26,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Wiegand reader: %v\n", err)
		os.Exit(1)
	}
	defer reader4.Close()

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Keep program running to process Wiegand data
	<-ctx.Done()
	fmt.Println("Shutting down Wiegand reader")
}
