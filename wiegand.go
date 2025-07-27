// Package wiegand provides a thread-safe library for reading Wiegand protocol data
// from Raspberry Pi GPIO pins. It supports configurable D0 and D1 pins and delivers
// received data to a user-provided callback function as a string of digits.
package wiegand

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// Reader represents a Wiegand reader instance, managing GPIO pins and data collection.
type Reader struct {
	d0, d1   gpio.PinIO         // GPIO pins for Wiegand D0 and D1
	data     []byte             // Buffer for collecting Wiegand bits
	mu       sync.Mutex         // Protects data buffer
	callback func(string)       // Callback to receive Wiegand data as digits
	ctx      context.Context    // Context for cancellation
	cancel   context.CancelFunc // Cancels the reader
	timeout  time.Duration      // Timeout for detecting end of Wiegand frame
	maxBits  int                // Maximum bits to collect (e.g., 26 for standard Wiegand)
}

// Config holds configuration for creating a new Wiegand Reader.
type Config struct {
	D0Pin, D1Pin string        // GPIO pin names (e.g., "GPIO14", "GPIO15")
	Callback     func(string)  // Function to receive Wiegand data
	Timeout      time.Duration // Timeout for frame completion (default 100ms)
	MaxBits      int           // Maximum bits per frame (default 26)
}

// DefaultTimeout is the default duration to wait for a complete Wiegand frame.
const DefaultTimeout = 100 * time.Millisecond

// DefaultMaxBits is the default maximum number of bits for a Wiegand frame.
const DefaultMaxBits = 26

// New creates a new Wiegand Reader for the specified D0 and D1 GPIO pins.
// It initializes the GPIO pins, sets up interrupt handling, and starts a goroutine
// to process Wiegand data. The callback function receives the Wiegand data as a
// string of digits. The Reader is thread-safe and can be stopped using the context.
func New(ctx context.Context, cfg Config) (*Reader, error) {
	// Initialize periph.io host
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize periph host: %w", err)
	}

	// Validate configuration
	if cfg.D0Pin == "" || cfg.D1Pin == "" {
		return nil, errors.New("D0Pin and D1Pin must be specified")
	}
	if cfg.Callback == nil {
		return nil, errors.New("callback function must be provided")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.MaxBits <= 0 {
		cfg.MaxBits = DefaultMaxBits
	}

	// Look up GPIO pins
	d0 := gpioreg.ByName(cfg.D0Pin)
	d1 := gpioreg.ByName(cfg.D1Pin)
	if d0 == nil || d1 == nil {
		return nil, fmt.Errorf("invalid GPIO pins: D0=%s, D1=%s", cfg.D0Pin, cfg.D1Pin)
	}

	// Configure pins as inputs with pull-down resistors (for optocoupler output)
	if err := d0.In(gpio.PullDown, gpio.BothEdges); err != nil {
		return nil, fmt.Errorf("failed to configure D0 pin %s: %w", cfg.D0Pin, err)
	}
	if err := d1.In(gpio.PullDown, gpio.BothEdges); err != nil {
		return nil, fmt.Errorf("failed to configure D1 pin %s: %w", cfg.D1Pin, err)
	}

	// Create Reader
	r := &Reader{
		d0:       d0,
		d1:       d1,
		data:     make([]byte, 0, cfg.MaxBits),
		callback: cfg.Callback,
		timeout:  cfg.Timeout,
		maxBits:  cfg.MaxBits,
	}

	// Create cancellable context
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start goroutines for reading pins and processing data
	go r.watchPin(r.d0, 0)
	go r.watchPin(r.d1, 1)
	go r.processData()

	return r, nil
}

// watchPin monitors a GPIO pin for falling edges (Wiegand pulses via optocoupler) and sends bits to the data buffer.
func (r *Reader) watchPin(pin gpio.PinIO, bit byte) {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			// Wait for a rising edge (optocoupler pulls GPIO high when active)
			//fmt.Println("saw falling edge: ", pin)
			if pin.WaitForEdge(100 * time.Millisecond) && pin.Read() == gpio.High {
				r.mu.Lock()
				r.data = append(r.data, bit)
				r.mu.Unlock()
				fmt.Println("saw edge")
			}
		}
	}
}

// processData collects Wiegand bits, detects complete frames, and invokes the callback.
func (r *Reader) processData() {
	ticker := time.NewTicker(r.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			if len(r.data) > 0 && len(r.data) <= r.maxBits {
				// Convert bits to decimal string
				data := bitsToDecimal(r.data)
				r.data = r.data[:0] // Reset buffer
				r.mu.Unlock()
				// Invoke callback in a separate goroutine to avoid blocking
				go r.callback(data)
			} else {
				// Clear buffer if too many bits or empty
				r.data = r.data[:0]
				r.mu.Unlock()
			}
		}
	}
}

// bitsToDecimal converts a slice of bits (0s and 1s) to a decimal string.
func bitsToDecimal(bits []byte) string {
	var num uint64
	for _, bit := range bits {
		num = (num << 1) | uint64(bit)
	}
	return fmt.Sprintf("%d", num)
}

// Close stops the Wiegand reader and releases resources.
func (r *Reader) Close() error {
	r.cancel()
	return nil
}
