package processors

import (
	"time"

	"github.com/alsosee/finder/structs"
)

type SearchIndexer struct {
	Host      string
	APIKey    string
	MasterKey string
	IndexName string
	StateFile string
	Force     string
	Timeout   time.Duration
}

var _ structs.Processor = (*SearchIndexer)(nil)

func (s *SearchIndexer) Init() error {
	return nil
}

func (s *SearchIndexer) ProcessFile() error {
	return nil
}

func (s *SearchIndexer) ProcessDirectory() error {
	return nil
}

func (s *SearchIndexer) Finalize() error {
	return nil
}
