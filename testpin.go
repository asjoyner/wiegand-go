package wiegand

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/pin"
	"periph.io/x/host/v3"
)

// TestPinEdge initializes the specified GPIO pins (or all "free" pins if none specified),
// prints their initial states, and continuously monitors for edge transitions until interrupted by Ctrl+C.
func TestPinEdge(pinNames []string) {
	// Initialize the periph host
	if _, err := host.Init(); err != nil {
		log.Fatalf("Failed to initialize periph host: %v", err)
	}

	// Define typically reserved pins for Raspberry Pi 5
	reservedPins := map[string]struct{}{
		"GPIO0":  {}, // ID_SD (HAT EEPROM)
		"GPIO1":  {}, // ID_SC (HAT EEPROM)
		"GPIO2":  {}, // I2C1_SDA
		"GPIO3":  {}, // I2C1_SCL
		"GPIO7":  {}, // SPI0_CE1_N
		"GPIO8":  {}, // SPI0_CE0_N
		"GPIO9":  {}, // SPI0_MISO
		"GPIO10": {}, // SPI0_MOSI
		"GPIO11": {}, // SPI0_SCLK
		"GPIO14": {}, // UART0_TXD
		"GPIO15": {}, // UART0_RXD
		// Optionally include GPIO22–GPIO27 if SDIO is a concern
		// "GPIO22": {}, // SDIO potential
		// "GPIO23": {},
		// "GPIO24": {},
		// "GPIO25": {},
		// "GPIO26": {},
		// "GPIO27": {},
	}

	// Determine which pins to monitor
	var pins []gpio.PinIO
	var monitoredPins []string

	if len(pinNames) == 0 {
		// If no pins specified, use all "free" GPIO pins (excluding reserved)
		for _, gpioPin := range gpioreg.All() {
			// Skip non-GPIO pins (e.g., power, ground) and reserved pins
			if _, reserved := reservedPins[gpioPin.Name()]; reserved || strings.HasPrefix(gpioPin.Name(), "3.3V") || strings.HasPrefix(gpioPin.Name(), "5V") || strings.HasPrefix(gpioPin.Name(), "GND") {
				// Attempt to get pin function for logging
				var funcName string
				if pf, ok := gpioPin.(pin.PinFunc); ok {
					funcName = string(pf.Func())
				} else {
					funcName = "unknown"
				}
				log.Printf("Skipping reserved or non-GPIO pin %s (function: %s)", gpioPin.Name(), funcName)
				continue
			}
			// Check if pin is configured for an alternate function (I2C, SPI, UART, SDIO)
			if pf, ok := gpioPin.(pin.PinFunc); ok {
				if funcName := string(pf.Func()); strings.Contains(funcName, "I2C") || strings.Contains(funcName, "SPI") || strings.Contains(funcName, "UART") || strings.Contains(funcName, "SDIO") {
					log.Printf("Skipping pin %s with alternate function: %s", gpioPin.Name(), funcName)
					continue
				}
			}
			pins = append(pins, gpioPin)
			monitoredPins = append(monitoredPins, gpioPin.Name())
		}
		if len(pins) == 0 {
			log.Fatal("No free GPIO pins available")
		}
		fmt.Printf("Monitoring free GPIO pins: %s\n", strings.Join(monitoredPins, ", "))
	} else {
		// Process the provided pin names
		for _, name := range pinNames {
			gpioPin := gpioreg.ByName(name)
			if gpioPin == nil {
				log.Printf("Invalid GPIO pin: %s", name)
				continue
			}
			// Log the pin's function for transparency
			var funcName string
			if pf, ok := gpioPin.(pin.PinFunc); ok {
				funcName = string(pf.Func())
			} else {
				funcName = "unknown"
			}
			log.Printf("Selected pin %s (function: %s)", gpioPin.Name(), funcName)
			pins = append(pins, gpioPin)
			monitoredPins = append(monitoredPins, name)
		}
		if len(pins) == 0 {
			log.Fatal("No valid GPIO pins to monitor")
		}
		fmt.Printf("Monitoring pins: %s\n", strings.Join(monitoredPins, ", "))
	}

	// Create a channel to signal goroutines to stop
	stopCh := make(chan struct{})

	// Launch a goroutine for each pin to monitor its state and transitions
	for _, gpioPin := range pins {
		go monitorPin(gpioPin, stopCh)
	}

	// Set up signal handling for Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh // Wait for the interrupt signal

	// Signal all goroutines to stop
	close(stopCh)

	// Allow a brief moment for goroutines to exit cleanly
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Shutting down")
}

// monitorPin configures a pin, prints its initial state, and continuously checks for edge transitions.
func monitorPin(gpioPin gpio.PinIO, stopCh <-chan struct{}) {
	// Configure the pin as input with pull-down resistor, detecting both rising and falling edges
	if err := gpioPin.In(gpio.PullDown, gpio.BothEdges); err != nil {
		log.Printf("Failed to configure pin %s: %v", gpioPin, err)
		return
	}

	// Print the initial state of the pin
	initialLevel := gpioPin.Read()
	fmt.Printf("Pin %s initial state: %s\n", gpioPin, initialLevel)

	// Continuously monitor for edge transitions until stopped
	for {
		select {
		case <-stopCh:
			return
		default:
			// Wait for an edge with a timeout to allow checking the stop channel
			if gpioPin.WaitForEdge(100 * time.Millisecond) {
				level := gpioPin.Read()
				fmt.Printf("Edge detected on pin %s: %s\n", gpioPin, level)
			}
		}
	}
}
