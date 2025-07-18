package wiegand

import (
	"context"
	"testing"
	"time"
)

func TestNewReader(t *testing.T) {
	// Mock callback
	callback := func(data string) {
		// pass
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use invalid pins to simulate Raspberry Pi GPIO (since tests may not run on Pi)
	cfg := Config{
		D0Pin:    "GPIO_INVALID",
		D1Pin:    "GPIO_INVALID",
		Callback: callback,
		Timeout:  50 * time.Millisecond,
		MaxBits:  26,
	}

	reader, err := New(ctx, cfg)
	if err == nil {
		reader.Close()
		t.Fatal("expected error for invalid GPIO pins")
	}
	if err.Error() != "invalid GPIO pins: D0=GPIO_INVALID, D1=GPIO_INVALID" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBitsToDecimal(t *testing.T) {
	tests := []struct {
		bits     []byte
		expected string
	}{
		{[]byte{0, 1, 0, 1}, "5"},
		{[]byte{1, 1, 1, 1}, "15"},
		{[]byte{}, "0"},
	}

	for _, tt := range tests {
		result := bitsToDecimal(tt.bits)
		if result != tt.expected {
			t.Errorf("bitsToDecimal(%v) = %s; want %s", tt.bits, result, tt.expected)
		}
	}
}
