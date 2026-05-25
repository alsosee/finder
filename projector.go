package main

import (
	"fmt"
	"log"
)

type Projector interface {
	Name() string
	Run(*BuildGraph) error
}

func RunProjectors(graph *BuildGraph, projectors ...Projector) error {
	for _, projector := range projectors {
		log.Printf("Running %s projector", projector.Name())
		if err := projector.Run(graph); err != nil {
			return fmt.Errorf("running %s projector: %w", projector.Name(), err)
		}
	}
	return nil
}
