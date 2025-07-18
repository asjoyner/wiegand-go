package main

import (
	"flag"
	"strings"

	"github.com/asjoyner/wiegand-go"
)

func main() {
	// Define a flag for GPIO pins, defaulting to an empty string
	pinsFlag := flag.String("pins", "", "Comma-separated list of GPIO pins to test (e.g., GPIO4,GPIO17). If empty, all pins are used.")
	flag.Parse()

	// Parse the comma-separated list of pins
	var pinNames []string
	if *pinsFlag != "" {
		pinNames = strings.Split(*pinsFlag, ",")
		for i, name := range pinNames {
			pinNames[i] = strings.TrimSpace(name) // Remove any whitespace
		}
	}

	// Run the pin monitoring logic
	wiegand.TestPinEdge(pinNames)
}
