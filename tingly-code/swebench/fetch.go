package swebench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomlx/go-huggingface/hub"
	parquet "github.com/parquet-go/parquet-go"
)

// SWEbenchParquetEntry represents a row in SWE-bench parquet file
type SWEbenchParquetEntry struct {
	InstanceID             string `parquet:"instance_id,optional"`
	Repo                   string `parquet:"repo,optional"`
	BaseCommit             string `parquet:"base_commit,optional"`
	Patch                  string `parquet:"patch,optional"`
	TestPatch              string `parquet:"test_patch,optional"`
	ProblemStatement       string `parquet:"problem_statement,optional"`
	HintsText              string `parquet:"hints_text,optional"`
	CreatedAt              string `parquet:"created_at,optional"`
	Version                string `parquet:"version,optional"`
	FailToPass             string `parquet:"FAIL_TO_PASS,optional"` // JSON string
	PassToPass             string `parquet:"PASS_TO_PASS,optional"` // JSON string
	EnvironmentSetupCommit string `parquet:"environment_setup_commit,optional"`
}

// Fetcher handles downloading SWEbench data from Hugging Face
type Fetcher struct {
	cacheDir string
}

// NewFetcher creates a new SWEbench data fetcher
func NewFetcher(cacheDir string) *Fetcher {
	return &Fetcher{
		cacheDir: cacheDir,
	}
}

// DatasetType represents the type of SWEbench dataset
type DatasetType string

const (
	DatasetTypeFull     DatasetType = "full"
	DatasetTypeLite     DatasetType = "lite"
	DatasetTypeVerified DatasetType = "verified"
)

// FetchOptions controls how data is fetched
type FetchOptions struct {
	// Dataset is which variant to download
	Dataset DatasetType

	// ForceDownload forces re-download even if cached
	ForceDownload bool

	// OutputPath is where to save the data (overrides default)
	OutputPath string

	// HFToken is optional HuggingFace auth token
	HFToken string

	// Progress reports download progress
	Progress func(msg string)
}

// getCachePath returns the cache file path for a dataset type
func (f *Fetcher) getCachePath(dataset DatasetType) string {
	_, filename := f.getRepoAndFilename(dataset)
	jsonName := strings.Replace(filepath.Base(filename), ".parquet", ".json", 1)
	return filepath.Join(f.cacheDir, string(dataset), jsonName)
}

// getRepoAndFilename returns the HuggingFace repo and filename for a dataset type
func (f *Fetcher) getRepoAndFilename(dataset DatasetType) (repoID, filename string) {
	switch dataset {
	case DatasetTypeLite:
		return "princeton-nlp/SWE-bench_Lite", "data/test-00000-of-00001.parquet"
	case DatasetTypeVerified:
		return "princeton-nlp/SWE-bench_Verified", "data/test-00000-of-00001.parquet"
	default:
		return "princeton-nlp/SWE-bench", "data/test-00000-of-00001.parquet"
	}
}

// Fetch downloads the SWEbench dataset using go-huggingface
func (f *Fetcher) Fetch(opts FetchOptions) (*TaskSet, error) {
	repoID, filename := f.getRepoAndFilename(opts.Dataset)

	// Determine output path
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = f.getCachePath(opts.Dataset)
	}

	// Check if already cached
	if !opts.ForceDownload {
		if cached, err := f.loadCached(outputPath); cached != nil {
			return cached, nil
		} else if err != nil {
			// Cache miss or invalid, continue to download
		}
	}

	// Create HuggingFace repo
	hfRepo := hub.New(repoID).WithType(hub.RepoTypeDataset)
	if opts.HFToken != "" {
		hfRepo = hfRepo.WithAuth(opts.HFToken)
	}

	// Download from HuggingFace
	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Fetching %s from HuggingFace...", filename))
	}

	downloadedFiles, err := hfRepo.DownloadFiles(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to download from HuggingFace: %w", err)
	}

	if len(downloadedFiles) == 0 {
		return nil, fmt.Errorf("no files downloaded")
	}

	localPath := downloadedFiles[0]

	// Read and parse the parquet file
	set, err := f.parseParquetFile(localPath, opts.Progress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parquet file: %w", err)
	}

	// Add metadata
	set.Version = string(opts.Dataset)
	set.Source = fmt.Sprintf("hf://%s/%s", repoID, filename)
	set.DownloadedAt = time.Now().Format(time.RFC3339)

	// Save to cache location as JSON for faster subsequent loading
	if err := f.saveToCache(outputPath, set); err != nil {
		return nil, fmt.Errorf("failed to save to cache: %w", err)
	}

	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Downloaded %d tasks", len(set.Tasks)))
	}

	return set, nil
}

// parseParquetFile reads a parquet file and converts it to TaskSet
func (f *Fetcher) parseParquetFile(filePath string, progress func(msg string)) (*TaskSet, error) {
	fReader, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fReader.Close()

	fSize, err := fReader.Stat()
	if err != nil {
		return nil, err
	}

	fParquet, err := parquet.OpenFile(fReader, fSize.Size())
	if err != nil {
		return nil, err
	}

	reader := parquet.NewGenericReader[SWEbenchParquetEntry](fParquet)
	defer reader.Close()

	// Read all rows
	var tasks []Task
	const batchSize = 100
	batch := make([]SWEbenchParquetEntry, batchSize)

	for {
		n, err := reader.Read(batch)
		if n == 0 {
			break
		}

		for i := 0; i < n; i++ {
			entry := batch[i]
			task := f.parquetEntryToTask(entry)
			tasks = append(tasks, task)
		}

		if progress != nil {
			progress(fmt.Sprintf("Parsed %d tasks...", len(tasks)))
		}

		if err != nil {
			break
		}
	}

	return &TaskSet{Tasks: tasks}, nil
}

// parquetEntryToTask converts a parquet entry to a Task
func (f *Fetcher) parquetEntryToTask(entry SWEbenchParquetEntry) Task {
	taskID := entry.InstanceID
	if taskID == "" {
		// Generate task ID from repo and commit
		taskID = fmt.Sprintf("%s__%s", strings.ReplaceAll(entry.Repo, "/", "_"), entry.BaseCommit[:8])
	}

	return Task{
		TaskID:           taskID,
		Repo:             entry.Repo,
		Version:          entry.Version,
		BaseCommit:       entry.BaseCommit,
		ProblemStatement: entry.ProblemStatement,
		Hints:            parseHints(entry.HintsText),
		CreatedAt:        entry.CreatedAt,
		TestCommand:      "pytest", // Default for Python projects
		EnvironmentSetup: entry.EnvironmentSetupCommit,
	}
}

// parseHints parses the hints_text field into a slice of hints
func parseHints(hintsText string) []string {
	if hintsText == "" {
		return nil
	}
	// Hints are often separated by newlines or numbered
	hints := strings.Split(hintsText, "\n")
	var result []string
	for _, h := range hints {
		h = strings.TrimSpace(h)
		if h != "" && !strings.HasPrefix(h, "###") && !strings.HasPrefix(h, "#") {
			// Remove numbering like "1.", "2.", etc.
			h = strings.TrimLeft(h, "0123456789.) ")
			result = append(result, h)
		}
	}
	return result
}

// loadCached tries to load a cached dataset
func (f *Fetcher) loadCached(path string) (*TaskSet, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // Not cached
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var set TaskSet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, err
	}

	fmt.Printf("Using cached dataset: %s (%d tasks)\n", path, len(set.Tasks))

	return &set, nil
}

// saveToCache saves the dataset to cache
func (f *Fetcher) saveToCache(path string, set *TaskSet) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ListTasks lists all tasks from a cached dataset
func (f *Fetcher) ListTasks(dataset DatasetType) ([]string, error) {
	path := f.getCachePath(dataset)

	set, err := f.loadCached(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load cached dataset: %w", err)
	}
	if set == nil {
		return nil, fmt.Errorf("dataset not cached. Run 'download' first")
	}

	return set.ListTasks(), nil
}

// GetTask gets a specific task from the cached dataset
func (f *Fetcher) GetTask(taskID string, dataset DatasetType) (*Task, error) {
	path := f.getCachePath(dataset)

	set, err := f.loadCached(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load cached dataset: %w", err)
	}
	if set == nil {
		return nil, fmt.Errorf("dataset not cached. Run 'download' first")
	}

	task, found := set.FindTask(taskID)
	if !found {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// GetCachedPath returns the cache path for a dataset
func (f *Fetcher) GetCachedPath(dataset DatasetType) string {
	return f.getCachePath(dataset)
}
