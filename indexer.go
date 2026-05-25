package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"

	"github.com/alsosee/finder/structs"
)

type searchClient interface {
	Index(uid string) searchIndex
	GetTask(taskID int64) (*meilisearch.Task, error)
}

type searchIndex interface {
	AddDocumentsInBatches(documentsPtr interface{}, batchSize int, primaryKey ...string) ([]meilisearch.TaskInfo, error)
	DeleteDocuments(identifiers []string) (*meilisearch.TaskInfo, error)
}

type meiliSearchClient struct {
	client meilisearch.ServiceManager
}

func (c meiliSearchClient) Index(uid string) searchIndex {
	return c.client.Index(uid)
}

func (c meiliSearchClient) GetTask(taskID int64) (*meilisearch.Task, error) {
	return c.client.GetTask(taskID)
}

type indexUpdatePlan struct {
	deleteIDs   []string
	updatePaths []string
}

// Indexer reads files and writes them to a MeiliSearch index.
type Indexer struct {
	client searchClient
	state  map[string]string
	graph  *BuildGraph
}

// NewIndexer creates a new Indexer.
func NewIndexer(
	client searchClient,
	graph *BuildGraph,
) *Indexer {
	return &Indexer{
		client: client,
		state:  graph.Hashes,
		graph:  graph,
	}
}

// Index writes graph documents to MeiliSearch.
func (i *Indexer) Index(stateFile, index, force string) error {
	state, err := readStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}

	log.Printf("State file contains %d entries", len(state))

	if err := i.updateIndex(state, index, force); err != nil {
		return fmt.Errorf("updating index: %w", err)
	}

	if err := writeStateToFile(stateFile, i.state); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

func (i *Indexer) updateIndex(oldState map[string]string, index, force string) error {
	plan, err := i.planUpdate(oldState, force)
	if err != nil {
		return err
	}

	if err := i.deleteFromIndex(plan.deleteIDs, index); err != nil {
		return fmt.Errorf("deleting documents: %w", err)
	}

	if err := i.addToIndex(plan.updatePaths, index); err != nil {
		return fmt.Errorf("adding documents: %w", err)
	}

	return nil
}

func (i *Indexer) planUpdate(oldState map[string]string, force string) (indexUpdatePlan, error) {
	if force == "all" {
		return indexUpdatePlan{updatePaths: sortedKeys(i.state)}, nil
	}

	if force != "" {
		forceList, err := parseForceList(force)
		if err != nil {
			return indexUpdatePlan{}, err
		}
		return indexUpdatePlan{updatePaths: i.expandThumbnailUpdates(forceList)}, nil
	}

	plan := indexUpdatePlan{}
	for path := range oldState {
		if _, ok := i.state[path]; !ok {
			plan.deleteIDs = append(plan.deleteIDs, searchDocumentIDForPath(path))
		}
	}

	var changed []string
	for path, hash := range i.state {
		if oldHash, ok := oldState[path]; !ok || oldHash != hash {
			changed = append(changed, path)
		}
	}
	plan.updatePaths = i.expandThumbnailUpdates(changed)
	sort.Strings(plan.deleteIDs)

	return plan, nil
}

func parseForceList(force string) ([]string, error) {
	var forceList []string
	if strings.HasPrefix(force, "[") {
		if err := json.Unmarshal([]byte(force), &forceList); err != nil {
			return nil, fmt.Errorf("parsing force list: %w", err)
		}
		return cleanPathList(forceList), nil
	}

	return cleanPathList(strings.Split(force, ",")), nil
}

func cleanPathList(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			result = append(result, path)
		}
	}
	return result
}

func (i *Indexer) expandThumbnailUpdates(paths []string) []string {
	updateSet := map[string]struct{}{}
	for _, path := range paths {
		updateSet[path] = struct{}{}
		for _, sharedPath := range i.graph.Media.PathsSharingThumb(path) {
			if _, exists := i.graph.Hashes[sharedPath]; exists {
				updateSet[sharedPath] = struct{}{}
			}
		}
	}

	return sortedKeys(updateSet)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func searchDocumentIDForPath(path string) string {
	return reSearchDocumentID.ReplaceAllString(removeFileExtention(path), "_")
}

var reSearchDocumentID = regexp.MustCompile("[^a-zA-Z0-9-_]")

func (i *Indexer) deleteFromIndex(ids []string, index string) error {
	if len(ids) == 0 {
		return nil
	}

	log.Printf("Deleting %d documents from index", len(ids))

	task, err := i.client.Index(index).DeleteDocuments(ids)
	if err != nil {
		return err
	}

	err = i.waitForTask(task.TaskUID, time.Minute*2)
	if err != nil {
		return fmt.Errorf("waiting for task %q: %w", task.TaskUID, err)
	}

	return nil
}

func (i *Indexer) addToIndex(paths []string, index string) error {
	documents := []*structs.Content{}

	for _, path := range paths {
		document, ok := i.graph.Document(path)
		if !ok {
			log.Printf("Document %q not found in graph, skipping", path)
			continue
		}
		documents = append(documents, document)
	}

	return i.addDocumentsToIndex(documents, index)
}

func (i *Indexer) addDocumentsToIndex(documents []*structs.Content, index string) error {
	if len(documents) == 0 {
		log.Printf("No documents to add to index %q", index)
		return nil
	}

	log.Printf("Adding %d documents to index %q", len(documents), index)
	for _, document := range documents {
		log.Printf("  %s", document.Source)
	}

	tasks, err := i.client.Index(index).AddDocumentsInBatches(documents, 100, "ID")
	if err != nil {
		return err
	}

	for _, task := range tasks {
		err := i.waitForTask(task.TaskUID, time.Minute*2)
		if err != nil {
			return fmt.Errorf("waiting for task %q: %w", task.TaskUID, err)
		}
	}
	return err
}

func (i *Indexer) waitForTask(taskID int64, timeout time.Duration) error {
	var (
		task  *meilisearch.Task
		err   error
		delay = time.Second
	)

	for {
		task, err = i.client.GetTask(taskID)
		if err != nil {
			return err
		}

		if task.Status == "succeeded" || task.Status == "failed" {
			break
		}

		time.Sleep(delay)

		// check timeout
		timeout -= delay
		if timeout <= 0 {
			return fmt.Errorf("task timeout")
		}

		log.Printf("Task %d status: %s", taskID, task.Status)
	}

	if task.Status == "failed" {
		return fmt.Errorf("task failed: %s", task.Error)
	}

	return nil
}

func readStateFromFile(stateFile string) (map[string]string, error) {
	state := make(map[string]string)

	f, err := os.Open(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("opening state file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid state file format")
		}
		relPath := parts[0]
		hash := parts[1]
		state[relPath] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	return state, nil
}

func writeStateToFile(stateFile string, state map[string]string) error {
	f, err := os.Create(stateFile)
	if err != nil {
		return fmt.Errorf("creating state file: %w", err)
	}
	defer f.Close()

	stateSlice := make([]string, 0, len(state))

	for relPath, hash := range state {
		stateSlice = append(stateSlice, fmt.Sprintf("%s\t%s", relPath, hash))
	}

	sort.Strings(stateSlice)

	for _, line := range stateSlice {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("writing state file: %w", err)
		}
	}

	return nil
}
