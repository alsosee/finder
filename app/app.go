// Description: Core application logic for processing files and generating content.
// App struct manages configuration, directories, processors, and the workflow for processing files.
// Main goal of the App is to read files from the info and media directories, and pass them through a series of processors to generate the final output.
package app

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	gitignore "github.com/sabhiram/go-gitignore"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

type App struct {

	// config is a site configuration, e.g. title, description, etc.
	// It is different from Config struct in main package,
	// which is used to store command line flags.
	Config structs.Config

	// A set of filesystems to process.
	FSS []fs.FS

	MediaDirectory string

	NumWorkers int

	processors []structs.ProcessorTodo

	eventBus chan structs.Event // Event bus for communication between processors

	ignorer *gitignore.GitIgnore

	dirContents map[string][]structs.File
	contents    structs.Contents

	// hashes is map of CRC32 hashes for each file.
	// key is a file path, value is a hash.
	// Used by indexer to check if file was changed.
	hashes map[string]string

	connections          structs.Connections         // Connections between contents
	mediaDirContents     map[string][]structs.Media  // Media files in directories, used by Index
	chainPages           map[string]map[bool]string  // Chain pages
	awardsMissingContent map[string][]structs.Award  // Awards with missing content
	missingContent       map[string]*structs.Content // map of the virtual paths to generated Content structs

	muDir      sync.Mutex // protects writes to dirContents
	muContents sync.Mutex // protects writes to contents
	muHashes   sync.Mutex // protects writes to hashes
}

func NewApp(
	config structs.Config,
	fss ...fs.FS,
) (*App, error) {
	// overrideConfig(&config)

	return &App{
		Config: config,
		FSS:    fss,
		// MediaDirectory: mediaDirectory,

		processors:           []structs.ProcessorTodo{},
		hashes:               map[string]string{},
		contents:             structs.Contents{},
		dirContents:          map[string][]structs.File{},
		connections:          structs.Connections{},
		mediaDirContents:     map[string][]structs.Media{},
		chainPages:           map[string]map[bool]string{},
		awardsMissingContent: map[string][]structs.Award{},
		missingContent:       map[string]*structs.Content{},
	}, nil
}

func (a *App) AddProcessor(p structs.ProcessorTodo) {
	a.processors = append(a.processors, p)
}

func (a *App) Run() error {
	// for _, processor := range a.processors {
	// 	if err := processor.Init(); err != nil {
	// 		return err
	// 	}
	// }

	// var g errgroup.Group
	// var done = make(chan struct{}, 1)

	type dirBatch struct {
		name      string
		hasThumbs bool
		media     []string
		files     []string
	}

	dirBatches := make(map[string]*dirBatch) // dir -> dir content

	for _, f := range a.FSS {
		// each filesystem can have its own ignore file
		ignorer, err := processIgnoreFile(f, ".finderignore")
		if err != nil {
			return fmt.Errorf("processing ignore file: %w", err)
		}

		err = fs.WalkDir(
			f,
			".",
			func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if ignorer.MatchesPath(path) {
					log.Printf("Ignoring %q", path)
					return nil
				}

				if entry.IsDir() {
					return nil
				}

				dir := filepath.Dir(path)
				batch, exists := dirBatches[dir]
				if !exists {
					batch = &dirBatch{name: dir}
					dirBatches[dir] = batch
				}

				ext := strings.ToLower(filepath.Ext(entry.Name()))
				switch ext {
				case ".yml", ".yaml":
					if entry.Name() == ".thumbs.yml" {
						batch.hasThumbs = true
						return nil
					}
					batch.files = append(batch.files, path)
				case ".jpg", ".jpeg", ".png", ".gif", ".webp":
					batch.media = append(batch.media, path)
				}

				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("scanning directory: %w", err)
		}
	}

	// Pipeline: process thumbnails → signal ready → process YAMLs
	readyDirs := make(chan *dirBatch, len(dirBatches))
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Producer: Process thumbnails concurrently, emit ready directories
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(readyDirs)

		var thumbWg sync.WaitGroup
		semaphore := make(chan struct{}, runtime.NumCPU()) // Limit concurrency

		for _, batch := range dirBatches {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if len(batch.media) == 0 {
				// No images, directory is immediately ready
				readyDirs <- batch
				continue
			}

			thumbWg.Add(1)
			semaphore <- struct{}{} // Acquire

			go func(b *dirBatch) {
				defer thumbWg.Done()
				defer func() { <-semaphore }() // Release

				// if err := a.createThumbnailSprite(b.dir, b.images); err != nil {
				// 	select {
				// 	case errChan <- fmt.Errorf("thumbnail sprite %s: %w", b.dir, err):
				// 		cancel()
				// 	default:
				// 	}
				// 	return
				// }

				select {
				case readyDirs <- b:
				case <-ctx.Done():
				}
			}(batch)
		}

		thumbWg.Wait()
	}()

	// Consumer: Process YAML files as directories become ready
	wg.Add(1)
	go func() {
		defer wg.Done()

		for batch := range readyDirs {
			for _, yamlPath := range batch.files {
				// todo: support multiple filesystems
				if err := a.processYAMLFile(a.FSS[0], yamlPath); err != nil {
					select {
					case errChan <- fmt.Errorf("yaml processing %s: %w", yamlPath, err):
						cancel()
					default:
					}
					return
				}
			}
		}
	}()

	wg.Wait()
	close(errChan)

	return <-errChan // Returns nil if no errors

	// for _, fs := range a.FSS {
	// 	g.Go(func() error {
	// 		return a.processFilesFromFS(fs, files)
	// 	})
	// }

	// a.addAwards()

	// // Generate missing files
	// m := a.missing()
	// a.addMissingFilesToPanels(m)

	// // Render Go templates
	// if err := a.generateGoTemplates(); err != nil {
	// 	return fmt.Errorf("generating go templates: %w", err)
	// }

	// g.processPanels()

	// if err := a.generateMissing(m); err != nil {
	// 	return fmt.Errorf("rendering missing: %w", err)
	// }

	// for _, processor := range a.processors {
	// 	if err := processor.ProcessFiles(a.contents); err != nil {
	// 		return fmt.Errorf("processing files %T: %w", processor, err)
	// 	}
	// }

	// for _, processor := range a.processors {
	// 	if err := processor.ProcessDirectories(a.dirContents); err != nil {
	// 		return fmt.Errorf("processing directories %T: %w", processor, err)
	// 	}
	// }

	// for _, processor := range a.processors {
	// 	if err := processor.Finalize(); err != nil {
	// 		return fmt.Errorf("finalizing processor %T: %w", processor, err)
	// 	}
	// }

	// Wait for processing to finish
	// go func() {
	// 	log.Printf("Waiting for processing to finish")
	// 	if err := g.Wait(); err != nil {
	// 		fmt.Printf("walking info directory: %s\n", err)
	// 	}

	// 	log.Printf("Processing finished")

	// 	close(done)
	// }()

	// <-done

	return nil
}

func processIgnoreFile(f fs.FS, path string) (*gitignore.GitIgnore, error) {
	if stat, err := fs.Stat(f, path); err == nil && !stat.IsDir() {
		b, err := fs.ReadFile(f, path)
		if err != nil {
			return nil, fmt.Errorf("reading %q file: %w", path, err)
		}

		return gitignore.CompileIgnoreLines(string(b)), nil
	}

	log.Printf("%q file not found, proceeding without it", path)
	return gitignore.CompileIgnoreLines(), nil
}

func (a *App) walkDirectory(files chan<- string) error {
	defer close(files)

	log.Printf("Walking directory")

	return fs.WalkDir(
		a.FSS[0],
		".",
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if a.ignorer.MatchesPath(path) {
				log.Printf("Ignoring %q", path)
				return nil
			}

			if entry.IsDir() {
				// a.addDir(path)
				return nil
			}

			files <- path
			return nil
		},
	)
}

// walkMediaDirectory scans the media directory for .thumbs.yml files,
// parses them and adds to g.mediaDirContents.
// mediaDirContents is a map where key is a directory path, and value is a list of media files in that directory.
// Information from .thumbs.yml used later in template to build links to thumbnails.
func (a *App) walkMediaDirectory() {
	if a.MediaDirectory == "" {
		log.Printf("No media files directory specified, skipping")
		return
	}

	mediaDir, err := filepath.Abs(a.MediaDirectory)
	if err != nil {
		log.Fatalf("Error getting absolute path for %q: %v", a.MediaDirectory, err)
	}

	log.Printf("Walking media directory %q", mediaDir)

	err = filepath.Walk(
		mediaDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// relPath := strings.TrimPrefix(path, mediaDir+string(filepath.Separator))

			if info.IsDir() {
				return nil
			}

			if info.Name() != ".thumbs.yml" {
				return nil
			}

			// media, err := structs.ParseMediaFile(path)
			// if err != nil {
			// 	return fmt.Errorf("parsing media file %q: %w", path, err)
			// }

			// g.addMedia(relPath, media)

			return nil
		},
	)

	if err != nil {
		log.Fatalf("Error walking media directory %q: %v", a.MediaDirectory, err)
	}

	log.Printf("Done walking media directory %q", a.MediaDirectory)
}

func (a *App) processFilesFromFS(f fs.FS, files <-chan string) error {
	var g errgroup.Group
	var done = make(chan struct{}, 1)
	var err error

	// collect errors from workers into a map for further annotation
	var muErrors sync.Mutex
	errorsMap := make(map[string]error)

	for i := 0; i < a.NumWorkers; i++ {
		g.Go(func() error {
			log.Printf("Worker %d started", i)
			for path := range files {
				err := a.processFile(f, path)
				if err != nil {
					muErrors.Lock()
					errorsMap[path] = err
					muErrors.Unlock()
				}
			}
			log.Printf("Worker %d finished", i)
			return nil
		})
	}

	go func() {
		err = g.Wait()
		close(done)
	}()

	<-done

	if len(errorsMap) > 0 {
		log.Printf("Errors occurred during file processing:")
		for path, e := range errorsMap {
			log.Printf("  %q: %v", path, e)
		}
	}

	return err
}

// processFile processes a single file.
// For content files, like YAML and Markdown, it adds Content struct to g.contents.
func (a *App) processFile(f fs.FS, path string) error {
	// log.Printf("Processing file %q", path)
	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		a.addFile(path)
		return a.processYAMLFile(f, path)
	// case ".gomd":
	// 	a.addFile(file)
	// 	return a.processGoMarkdownFile(file)
	// case ".md":
	// 	a.addFile(file)
	// 	return a.processMarkdownFile(file)
	// case ".jpeg", ".jpg", ".png":
	// 	a.addFile(file)
	// 	return a.processImageFile(file)
	// case ".mp4":
	// 	a.addFile(file)
	// 	return a.processVideoFile(file)
	default:
		// todo:
		// if file == "_redirects" {
		// return a.copyFileAsIs(file)
		// }
		return fmt.Errorf("unknown file type: %q", path)
	}
}

func (a *App) addDirContents(path string, file structs.File) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	a.muDir.Lock()
	a.dirContents[dir] = append(a.dirContents[dir], file)
	a.muDir.Unlock()
}

func (a *App) addFile(path string) {
	a.addDirContents(path, structs.File{
		Name: removeFileExtention(filepath.Base(path)),
		// Image: a.getImageForPath(removeFileExtention(path)),
	})
}

func (a *App) processYAMLFile(f fs.FS, file string) error {
	b, err := fs.ReadFile(f, file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	a.addHash(file, b)

	var content structs.Content
	if err = yaml.Unmarshal(b, &content); err != nil {
		return fmt.Errorf("unmarshaling yaml: %w", err)
	}

	content.Source = file
	content.SourceNoExtention = removeFileExtention(file)
	content.GenerateID()
	// content.AddMedia(a.getImageForPath)

	a.addContent(content)
	// g.addConnections(content)

	return nil
}

// func (a *App) processMarkdownFile(file string) error {
// 	b, err := os.ReadFile(filepath.Join(a.InfoDirectory, file))
// 	if err != nil {
// 		return fmt.Errorf("reading file: %w", err)
// 	}

// 	htmlBody := markdown.ToHTML(b, nil, nil)

// 	// replace [ ] and [x] with checkboxes and break lines with <br> at the end of the line with checkbox
// 	// except for the first line
// 	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[ ] "), []byte(`<br><input type="checkbox" disabled> `))
// 	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[x] "), []byte(`<br><input type="checkbox" disabled checked> `))
// 	htmlBody = bytes.ReplaceAll(htmlBody, []byte("<p><br>"), []byte("<p>"))

// 	g.addContent(structs.Content{
// 		Source: file,
// 		HTML:   string(htmlBody),
// 	})
// 	return nil
// }

// func (a *App) processGoMarkdownFile(file string) error {
// 	b, err := os.ReadFile(filepath.Join(a.InfoDirectory, file))
// 	if err != nil {
// 		return fmt.Errorf("reading file: %w", err)
// 	}

// 	g.addContent(structs.Content{
// 		Source: file,
// 		HTML:   string(b),
// 	})

// 	// conversion to HTML is done in generateGoTemplates()
// 	// after all the files are processed

// 	return nil
// }

// todo: should be part of htmlgen
// func (a *App) copyFileAsIs(file string) error {
// 	return copyFile(
// 		filepath.Join(a.InfoDirectory, file),
// 		filepath.Join(a.OutputDirectory, file),
// 	)
// }

func (a *App) overrideConfig(config *structs.Config) {
	// if a.MediaHost != "" {
	// 	config.MediaHost = a.MediaHost
	// }
	// if a.SearchHost != "" {
	// 	config.SearchHost = a.SearchHost
	// }
	// if a.SearchIndexName != "" {
	// 	config.SearchIndexName = a.SearchIndexName
	// }
	// if a.SearchAPIKey != "" {
	// 	config.SearchAPIKey = a.SearchAPIKey
	// }
}

func (a *App) addContent(content structs.Content) {
	a.muContents.Lock()
	a.contents[content.SourceNoExtention] = content
	a.muContents.Unlock()
}

func (a *App) addHash(path string, b []byte) {
	a.muHashes.Lock()
	defer a.muHashes.Unlock()

	a.hashes[path] = fmt.Sprintf("%x", crc32.ChecksumIEEE(b))
}

func removeFileExtention(path string) string {
	withoutExt := path[:len(path)-len(filepath.Ext(path))]
	if withoutExt != "" {
		return withoutExt
	}
	return path
}

func copyFile(src, dst string) error {
	log.Printf("Copying file %q to %q", src, dst)
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
