package processors

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/alsosee/finder/structs"
)

type Profile struct {
	CPUProfile string
	MemProfile string

	fileCPUProfile *os.File
}

var _ structs.Processor = (*Profile)(nil)

func (p *Profile) Init() error {
	if p.CPUProfile == "" || p.MemProfile == "" {
		return fmt.Errorf("cpu and memory profile paths must be set")
	}

	var err error
	p.fileCPUProfile, err = os.Create(p.CPUProfile)
	if err != nil {
		return fmt.Errorf("creating cpu profile: %v", err)
	}
	if err := pprof.StartCPUProfile(p.fileCPUProfile); err != nil {
		return fmt.Errorf("starting cpu profile: %v", err)
	}
	return nil
}

func (p *Profile) ProcessFile() error {
	return nil
}

func (p *Profile) ProcessDirectory() error {
	return nil
}

func (p *Profile) Finalize() error {
	// Stop CPU profiling and take a memory snapshot
	pprof.StopCPUProfile()
	if p.fileCPUProfile != nil {
		if err := p.fileCPUProfile.Close(); err != nil {
			return fmt.Errorf("closing cpu profile file: %v", err)
		}
	}

	f, err := os.Create(p.MemProfile)
	if err != nil {
		return fmt.Errorf("creating memory profile: %v", err)
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("writing memory profile: %v", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("closing memory profile: %v", err)
	}

	return nil
}
