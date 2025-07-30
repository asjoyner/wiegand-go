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

func TestDecodeBits(t *testing.T) {
	tests := []struct {
		name           string
		bits           []byte
		shift          int
		maskBits       int
		siteCodeStart  int
		siteCodeLength int
		wantTag        string
		wantSite       string
		wantErr        bool
	}{
		{
			name:           "Empty slice",
			bits:           []byte{},
			shift:          0,
			maskBits:       0,
			siteCodeStart:  0,
			siteCodeLength: 0,
			wantTag:        "",
			wantSite:       "",
			wantErr:        true,
		},
		{
			name:           "26-bit Wiegand with site code",
			bits:           []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1}, // Parity, Site=1, Tag=0x5555
			shift:          1,
			maskBits:       24,
			siteCodeStart:  1,
			siteCodeLength: 8,
			wantTag:        "21845", // 0x5555
			wantSite:       "0",     // Bits 1-8 = 00000000
			wantErr:        false,
		},
		{
			name: "26-bit Wiegand with high value tag",
			bits: []byte{1, 0, 0, 0, 0, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0},

			shift:          1,
			maskBits:       24,
			siteCodeStart:  1,
			siteCodeLength: 8,
			wantTag:        "999999",
			wantSite:       "15", // Bits 1-8 = 00000000
			wantErr:        false,
		},
		{
			name:           "34-bit Wiegand with larger site code",
			bits:           []byte{1, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0}, // Parity, Site=0x3333, Tag=0xCCCC
			shift:          1,
			maskBits:       32,
			siteCodeStart:  1,
			siteCodeLength: 16,
			wantTag:        "52428", // 0xCCCC
			wantSite:       "13107", // 0x3333
			wantErr:        false,
		},
		{
			name:           "37-bit Wiegand with larger site code",
			bits:           []byte{1, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0}, // Parity, Site=0x777, Tag=0xCCCC
			shift:          1,
			maskBits:       35,
			siteCodeStart:  1,
			siteCodeLength: 19,
			wantTag:        "35127298", // 0x21772A
			wantSite:       "967",      // 0x3C7
			wantErr:        false,
		},
		{
			name:           "Insufficient bits for mask",
			bits:           []byte{1, 0, 1},
			shift:          1,
			maskBits:       3,
			siteCodeStart:  0,
			siteCodeLength: 1,
			wantTag:        "",
			wantSite:       "",
			wantErr:        true,
		},
		{
			name:           "Invalid site code range",
			bits:           []byte{1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0},
			shift:          1,
			maskBits:       24,
			siteCodeStart:  25,
			siteCodeLength: 2,
			wantTag:        "",
			wantSite:       "",
			wantErr:        true,
		},
		{
			name:           "Invalid bit value",
			bits:           []byte{1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			shift:          1,
			maskBits:       24,
			siteCodeStart:  1,
			siteCodeLength: 8,
			wantTag:        "",
			wantSite:       "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, gotSite, err := decodeBits(tt.bits, tt.shift, tt.maskBits, tt.siteCodeStart, tt.siteCodeLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBits(%v, %d, %d, %d, %d) error = %v, wantErr %v", tt.bits, tt.shift, tt.maskBits, tt.siteCodeStart, tt.siteCodeLength, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if gotTag != tt.wantTag {
				t.Errorf("decodeBits(%v, %d, %d, %d, %d) tag = %s, want %s", tt.bits, tt.shift, tt.maskBits, tt.siteCodeStart, tt.siteCodeLength, gotTag, tt.wantTag)
			}
			if gotSite != tt.wantSite {
				t.Errorf("decodeBits(%v, %d, %d, %d, %d) site = %s, want %s", tt.bits, tt.shift, tt.maskBits, tt.siteCodeStart, tt.siteCodeLength, gotSite, tt.wantSite)
			}
		})
	}
}
