package main

import (
	"log"
	"time"
)

func measureTime() func() {
	start := time.Now()
	return func() {
		log.Printf("Elapsed: %v", time.Since(start))
	}
}
