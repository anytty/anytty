package main

import (
	"io"
	"log"
)

func configureIOSLogging() {
	log.SetFlags(0)
	log.SetOutput(io.Discard)
}
