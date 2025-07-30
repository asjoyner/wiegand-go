# wiegand-go

A Go package for reading Wiegand protocol data from GPIO pins on a Raspberry Pi 5, ideal for RFID card readers. Includes a `testpin` utility to verify GPIO connections. Uses `periph.io` (v3.8.5) for GPIO operations.

## Features

- Reads Wiegand data (e.g., 26-bit) from two GPIO pins (default: GPIO14/D0, GPIO15/D1).
- Thread-safe with mutexes and context cancellation.
- `testpin` command monitors GPIO edge transitions to verify hardware connections.
- Supports 817C optocouplers for 5V Wiegand signal isolation.
- Excludes reserved pins (GPIO0–3, 7–11, 14–15) and alternate functions (I2C, SPI, UART, SDIO).

## Requirements

- Go 1.21+
- Raspberry Pi 5 with Raspberry Pi OS (64-bit)
- Hardware: Wiegand device, 2x 817C optocouplers, 470Ω and 220–330Ω resistors, 1.5KE6.8CA TVS diodes
- Run with `sudo` for GPIO access

## Installation

1. Clone and install:
```bash
git clone https://github.com/asjoyner/wiegand-go.git
cd wiegand-go
go mod tidy
go build ./...
```

- Build `testpin`:
```bash
cd cmd/testpin
go build
```
## Usage
Wiegand Reader

See `wiegand/example/main.go:`

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/asjoyner/wiegand-go"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    cfg := wiegand.Config{
        D0Pin:    "GPIO14",
        D1Pin:    "GPIO15",
        Callback: func(data string) { fmt.Printf("Wiegand data: %s\n", data) },
        Timeout:  100 * time.Millisecond,
        MaxBits:  26,
    }
    reader, err := wiegand.New(ctx, cfg)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
        os.Exit(1)
    }
    defer reader.Close()
    select {}
}
```

Run:

```bash
cd cmd/example-usage/
go build
./example-usage
```

### Pin Testing (testpin)
- Monitor all free GPIO pins:
```bash
./testpin
```

Output:

```bash
Monitoring free GPIO pins: GPIO4,GPIO5,GPIO6,GPIO12,GPIO13,GPIO16,GPIO17,GPIO18,GPIO19,GPIO20,GPIO21,GPIO22,GPIO23,GPIO24,GPIO25,GPIO26,GPIO27
Pin GPIO4 initial state: Low
Edge detected on pin GPIO22: High
^C
Shutting down
```

- Monitor specific pins:

```bash
sudo ./testpin -pins=GPIO14,GPIO15
```

Output:

```bash
Monitoring pins: GPIO14,GPIO15
Pin GPIO14 initial state: Low
Pin GPIO15 initial state: Low
Edge detected on pin GPIO14: High
^C
Shutting down
```


## Testing Notes

Use `testpin` to verify Wiegand reader connections:
   - Connect Wiegand Reader:
      - Wire D0 (e.g., green wire) and D1 (e.g., white wire) to optocouplers, with emitters to GPIO14 (D0) and GPIO15 (D1).


Test Specific Pins:

```bash
sudo ./testpin -pins=GPIO14,GPIO15
```

   - Confirms edges on GPIO14 (D0) and GPIO15 (D1) when swiping a card.
   - If no edges, check wiring, resistor values, or TVS diode orientation.

Test All Free Pins:

```bash
sudo ./testpin
```

   - Identifies which pins receive Wiegand pulses if connections are uncertain.
   - Debugging:
      - Verify pulses (~40–100µs) with a multimeter or oscilloscope.
      - Check `/boot/firmware/config.txt` for conflicting pin settings (e.g., UART on GPIO14/15).
      - Run `pinctrl get 14-15` to ensure pins are set to `input`.
   - Confirm Pins:
      - Update wiegand.Config with verified pins (e.g., D0Pin: "GPIO14", D1Pin: "GPIO15").

