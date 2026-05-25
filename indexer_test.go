package main

import (
	"reflect"
	"testing"

	"github.com/alsosee/finder/structs"
	"github.com/meilisearch/meilisearch-go"
)

func TestIndexerPlanUpdateChangedDeletedAndThumbnailRelated(t *testing.T) {
	graph := testSearchGraph()
	indexer := NewIndexer(nil, graph)

	plan, err := indexer.planUpdate(map[string]string{
		"Movies/A.yml":    "old",
		"Movies/B.yml":    "same",
		"Movies/C.yml":    "same",
		"Movies/Gone.yml": "old",
	}, "")
	if err != nil {
		t.Fatalf("planUpdate() error = %v", err)
	}

	if want := []string{"Movies_Gone"}; !reflect.DeepEqual(plan.deleteIDs, want) {
		t.Fatalf("plan.deleteIDs = %v, want %v", plan.deleteIDs, want)
	}
	if want := []string{"Movies/A.yml", "Movies/B.yml"}; !reflect.DeepEqual(plan.updatePaths, want) {
		t.Fatalf("plan.updatePaths = %v, want %v", plan.updatePaths, want)
	}
}

func TestIndexerPlanUpdateForceModes(t *testing.T) {
	graph := testSearchGraph()
	indexer := NewIndexer(nil, graph)

	plan, err := indexer.planUpdate(nil, "all")
	if err != nil {
		t.Fatalf("planUpdate(all) error = %v", err)
	}
	if want := []string{"Movies/A.yml", "Movies/B.yml", "Movies/C.yml"}; !reflect.DeepEqual(plan.updatePaths, want) {
		t.Fatalf("planUpdate(all).updatePaths = %v, want %v", plan.updatePaths, want)
	}

	plan, err = indexer.planUpdate(nil, `["Movies/C.yml"]`)
	if err != nil {
		t.Fatalf("planUpdate(json force) error = %v", err)
	}
	if want := []string{"Movies/C.yml"}; !reflect.DeepEqual(plan.updatePaths, want) {
		t.Fatalf("planUpdate(json force).updatePaths = %v, want %v", plan.updatePaths, want)
	}

	plan, err = indexer.planUpdate(nil, "Movies/A.yml")
	if err != nil {
		t.Fatalf("planUpdate(comma force) error = %v", err)
	}
	if want := []string{"Movies/A.yml", "Movies/B.yml"}; !reflect.DeepEqual(plan.updatePaths, want) {
		t.Fatalf("planUpdate(comma force).updatePaths = %v, want %v", plan.updatePaths, want)
	}
}

func TestIndexerUpdateIndexUsesGraphDocumentsAndSearchIDs(t *testing.T) {
	graph := testSearchGraph()
	client := &fakeSearchClient{index: &fakeSearchIndex{}}
	indexer := NewIndexer(client, graph)

	err := indexer.updateIndex(map[string]string{
		"Movies/A.yml":    "old",
		"Movies/B.yml":    "same",
		"Movies/C.yml":    "same",
		"Movies/Gone.yml": "old",
	}, "info", "")
	if err != nil {
		t.Fatalf("updateIndex() error = %v", err)
	}

	if want := []string{"Movies_Gone"}; !reflect.DeepEqual(client.index.deletedIDs, want) {
		t.Fatalf("deleted IDs = %v, want %v", client.index.deletedIDs, want)
	}
	if want := []string{"Movies_A", "Movies_B"}; !reflect.DeepEqual(client.index.addedIDs, want) {
		t.Fatalf("added IDs = %v, want %v", client.index.addedIDs, want)
	}
}

func testSearchGraph() *BuildGraph {
	return &BuildGraph{
		Contents: structs.Contents{
			"Movies/A": {
				Source: "Movies/A.yml",
				Name:   "A",
			},
			"Movies/B": {
				Source: "Movies/B.yml",
				Name:   "B",
			},
			"Movies/C": {
				Source: "Movies/C.yml",
				Name:   "C",
			},
		},
		Media: MediaCatalog{
			"Movies": {
				{Path: "A.jpg", ThumbPath: "sheet.jpg"},
				{Path: "B.jpg", ThumbPath: "sheet.jpg"},
			},
		},
		Hashes: map[string]string{
			"Movies/A.yml": "new",
			"Movies/B.yml": "same",
			"Movies/C.yml": "same",
		},
	}
}

type fakeSearchClient struct {
	index *fakeSearchIndex
}

func (c *fakeSearchClient) Index(_ string) searchIndex {
	return c.index
}

func (c *fakeSearchClient) GetTask(_ int64) (*meilisearch.Task, error) {
	return &meilisearch.Task{Status: "succeeded"}, nil
}

type fakeSearchIndex struct {
	addedIDs   []string
	deletedIDs []string
}

func (i *fakeSearchIndex) AddDocumentsInBatches(documentsPtr interface{}, _ int, _ ...string) ([]meilisearch.TaskInfo, error) {
	documents := documentsPtr.([]*structs.Content)
	for _, document := range documents {
		i.addedIDs = append(i.addedIDs, document.ID)
	}
	return []meilisearch.TaskInfo{{TaskUID: 1}}, nil
}

func (i *fakeSearchIndex) DeleteDocuments(identifiers []string) (*meilisearch.TaskInfo, error) {
	i.deletedIDs = append(i.deletedIDs, identifiers...)
	return &meilisearch.TaskInfo{TaskUID: 2}, nil
}
