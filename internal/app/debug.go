package app

import (
	"log"
	"os"
)

func debugAppf(format string, args ...any) {
	if os.Getenv("PODJI_DEBUG_DATA") != "1" {
		return
	}
	log.Printf("podji:app "+format, args...)
}
