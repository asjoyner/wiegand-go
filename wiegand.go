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
	d0, d1      gpio.PinIO         // GPIO pins for Wiegand D0 and D1
	data        []byte             // Buffer for collecting Wiegand bits
	lastBitTime time.Time          // Time of the last received bit
	mu          sync.Mutex         // Protects data buffer and lastBitTime
	callback    func(string)       // Callback to receive Wiegand data as digits
	ctx         context.Context    // Context for cancellation
	cancel      context.CancelFunc // Cancels the reader
	timeout     time.Duration      // Timeout for detecting end of Wiegand frame
	maxBits     int                // Maximum bits to collect (e.g., 26 for standard Wiegand)
	pulse       chan bool          // Signals new pulse
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
func New(ctx context.Context, cfg Config) (*Reader, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize periph host: %w", err)
	}

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

	d0 := gpioreg.ByName(cfg.D0Pin)
	d1 := gpioreg.ByName(cfg.D1Pin)
	if d0 == nil || d1 == nil {
		return nil, fmt.Errorf("invalid GPIO pins: D0=%s, D1=%s", cfg.D0Pin, cfg.D1Pin)
	}

	if err := d0.In(gpio.PullDown, gpio.FallingEdge); err != nil {
		return nil, fmt.Errorf("failed to configure D0 pin %s: %w", cfg.D0Pin, err)
	}
	if err := d1.In(gpio.PullDown, gpio.FallingEdge); err != nil {
		return nil, fmt.Errorf("failed to configure D1 pin %s: %w", cfg.D1Pin, err)
	}

	r := &Reader{
		d0:       d0,
		d1:       d1,
		data:     make([]byte, 0, cfg.MaxBits),
		callback: cfg.Callback,
		timeout:  cfg.Timeout,
		maxBits:  cfg.MaxBits,
		pulse:    make(chan bool, 1), // Buffered to avoid blocking
	}

	r.ctx, r.cancel = context.WithCancel(ctx)

	go r.watchPin(r.d0, 0)
	go r.watchPin(r.d1, 1)
	go r.processData()

	return r, nil
}

// watchPin monitors a GPIO pin for falling edges and sends bits to the data buffer.
func (r *Reader) watchPin(pin gpio.PinIO, bit byte) {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			if pin.WaitForEdge(1*time.Second) && pin.Read() == gpio.Low { // Wait indefinitely for edge
				r.mu.Lock()
				r.data = append(r.data, bit)
				r.lastBitTime = time.Now()
				select {
				case r.pulse <- true:
				default:
				}
				r.mu.Unlock()
			}
		}
	}
}

// checkParity calculates even or odd parity for a range of bits in the data.
func checkParity(bits []byte, start, length int, even bool) bool {
	if start+length > len(bits) {
		return false
	}
	parity := 0
	for i := start; i < start+length; i++ {
		if bits[i] == 1 {
			parity++
		}
	}
	if even {
		return parity%2 == 0
	}
	return parity%2 == 1
}

// bitsToTag converts a slice of bits to a decimal string, applying shift and mask.
func bitsToTag(bits []byte, shift, maskBits int) string {
	var num uint64
	for _, bit := range bits {
		num = (num << 1) | uint64(bit)
	}
	num = (num >> uint(shift)) & ((1 << uint(maskBits)) - 1)
	return fmt.Sprintf("%d", num)
}

// processData collects Wiegand bits, detects complete frames, and invokes the callback.
func (r *Reader) processData() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.pulse:
			// Wait until timeout elapses since last bit
			for time.Since(r.lastBitTime) < r.timeout {
				select {
				case <-r.pulse:
					// New pulse received, reset timeout
				case <-r.ctx.Done():
					return
				case <-time.After(r.timeout - time.Since(r.lastBitTime)):
					// Timeout elapsed, process data
				}
			}
			r.mu.Lock()
			data := make([]byte, len(r.data)) // Copy data
			copy(data, r.data)
			r.data = r.data[:0] // Reset buffer
			r.mu.Unlock()

			if len(data) == 0 {
				continue
			}

			fmt.Printf("Received %d-bit value: %v\n", len(data), data)

			switch len(data) {
			case 26:
				tag := bitsToTag(data, 1, 24)
				if !checkParity(data, 0, 13, true) || !checkParity(data, 13, 13, false) {
					fmt.Printf("Invalid parity for 26-bit tag: %s\n", tag)
					continue
				}
				fmt.Printf("Received 26-bit tag: %s\n", tag)
				go r.callback(tag)
			case 34:
				tag := bitsToTag(data, 1, 32)
				if !checkParity(data, 0, 17, true) || !checkParity(data, 17, 17, false) {
					fmt.Printf("Invalid parity for 34-bit tag: %s\n", tag)
					continue
				}
				fmt.Printf("Received 34-bit tag: %s\n", tag)
				go r.callback(tag)
			case 37:
				tag := bitsToTag(data, 1, 35)
				if !checkParity(data, 0, 19, true) || !checkParity(data, 19, 18, false) {
					fmt.Printf("Invalid parity for 37-bit tag: %s\n", tag)
					continue
				}
				fmt.Printf("Received 37-bit tag: %s\n", tag)
				go r.callback(tag)
			default:
				fmt.Printf("Received unknown %d-bit value\n", len(data))
			}
		}
	}
}

// Close stops the Wiegand reader and releases resources.
func (r *Reader) Close() error {
	r.cancel()
	return nil
}
