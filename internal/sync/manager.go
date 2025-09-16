package sync

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openwebui-content-sync/internal/adapter"
	"github.com/openwebui-content-sync/internal/config"
	"github.com/openwebui-content-sync/internal/openwebui"
	"github.com/sirupsen/logrus"
)

// Manager handles synchronization between adapters and OpenWebUI
type Manager struct {
	openwebuiClient openwebui.ClientInterface
	storagePath     string
	knowledgeID     string
	fileIndex       map[string]*FileMetadata
	indexPath       string
}

// FileMetadata stores metadata about synced files
type FileMetadata struct {
	Path     string    `json:"path"`
	Hash     string    `json:"hash"`
	FileID   string    `json:"file_id"`
	Source   string    `json:"source"`
	SyncedAt time.Time `json:"synced_at"`
	Modified time.Time `json:"modified"`
}

// NewManager creates a new sync manager
func NewManager(openwebuiConfig config.OpenWebUIConfig, storageConfig config.StorageConfig) (*Manager, error) {
	client := openwebui.NewClient(openwebuiConfig.BaseURL, openwebuiConfig.APIKey)

	// Ensure storage directory exists
	if err := os.MkdirAll(storageConfig.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	indexPath := filepath.Join(storageConfig.Path, "file_index.json")

	manager := &Manager{
		openwebuiClient: client,
		storagePath:     storageConfig.Path,
		indexPath:       indexPath,
		fileIndex:       make(map[string]*FileMetadata),
	}

	// Load existing file index
	if err := manager.loadFileIndex(); err != nil {
		logrus.Warnf("Failed to load file index: %v", err)
	}

	return manager, nil
}

// SetKnowledgeID sets the knowledge ID for file operations
func (m *Manager) SetKnowledgeID(knowledgeID string) {
	logrus.Debugf("Setting knowledge ID: %s", knowledgeID)
	m.knowledgeID = knowledgeID
}

// InitializeFileIndex populates the file index with existing files from OpenWebUI
func (m *Manager) InitializeFileIndex(ctx context.Context) error {
	if m.knowledgeID == "" {
		logrus.Debugf("No knowledge ID set, skipping file index initialization")
		return nil
	}

	logrus.Info("Initializing file index from OpenWebUI knowledge base...")

	// Get files from the knowledge source
	files, err := m.openwebuiClient.GetKnowledgeFiles(ctx, m.knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to get files from knowledge source: %w", err)
	}

	logrus.Debugf("Found %d existing files in knowledge source", len(files))

	// Add files to index
	for _, file := range files {
		// Use filename as path if no path is available
		filePath := file.Path
		if filePath == "" {
			filePath = file.Meta.Name
		}

		// For existing files from OpenWebUI, use just the filename as the key
		// This avoids the "unknown:filename" issue and makes it easier to match
		// with files that will be synced from adapters
		fileKey := filePath

		// Use file ID as hash since OpenWebUI doesn't provide content hash
		// This means we won't detect content changes, but we can track file existence
		fileHash := file.ID
		if file.Hash != "" {
			fileHash = file.Hash
		}

		// Create file metadata
		metadata := &FileMetadata{
			Path:     filePath,
			Hash:     fileHash,
			FileID:   file.ID,
			Source:   "openwebui", // Mark as existing from OpenWebUI
			SyncedAt: time.Unix(file.CreatedAt, 0),
			Modified: time.Unix(file.UpdatedAt, 0),
		}

		m.fileIndex[fileKey] = metadata
		logrus.Debugf("Added existing file to index: %s (ID: %s, Hash: %s)", filePath, file.ID, fileHash)
	}

	logrus.Infof("File index initialized with %d existing files", len(files))

	// Save the updated index
	if err := m.saveFileIndex(); err != nil {
		logrus.Warnf("Failed to save initialized file index: %v", err)
	}

	return nil
}

// SyncFiles synchronizes files from adapters to OpenWebUI
func (m *Manager) SyncFiles(ctx context.Context, adapters []adapter.Adapter) error {
	logrus.Info("Starting file synchronization")

	// List available knowledge sources for debugging
	logrus.Debugf("Listing available knowledge sources...")
	knowledgeList, err := m.openwebuiClient.ListKnowledge(ctx)
	if err != nil {
		logrus.Warnf("Failed to list knowledge sources: %v", err)
	} else {
		logrus.Debugf("Available knowledge sources:")
		for _, knowledge := range knowledgeList {
			logrus.Debugf("  - ID: %s, Name: %s, Description: %s", knowledge.ID, knowledge.Name, knowledge.Description)
		}
	}

	// Track files that are currently present in repositories
	currentFiles := make(map[string]bool)

	for _, adpt := range adapters {
		logrus.Infof("Syncing files from adapter: %s", adpt.Name())

		files, err := adpt.FetchFiles(ctx)
		if err != nil {
			logrus.Errorf("Failed to fetch files from adapter %s: %v", adpt.Name(), err)
			continue
		}

		logrus.Debugf("Fetched %d files from adapter %s", len(files), adpt.Name())

		for _, file := range files {
			fileKey := fmt.Sprintf("%s:%s", adpt.Name(), file.Path)
			filenameKey := filepath.Base(file.Path) // Extract just the filename
			currentFiles[fileKey] = true
			currentFiles[filenameKey] = true // Also track by filename for existing files

			if err := m.syncFile(ctx, file, adpt.Name()); err != nil {
				logrus.Errorf("Failed to sync file %s: %v", file.Path, err)
				continue
			}
		}

		// Update last sync time
		adpt.SetLastSync(time.Now())
	}

	// Clean up orphaned files (files that are no longer in repositories)
	if err := m.cleanupOrphanedFiles(ctx, currentFiles); err != nil {
		logrus.Errorf("Failed to cleanup orphaned files: %v", err)
	}

	// Save updated file index
	if err := m.saveFileIndex(); err != nil {
		logrus.Errorf("Failed to save file index: %v", err)
	}

	logrus.Info("File synchronization completed")
	return nil
}

// syncFile synchronizes a single file
func (m *Manager) syncFile(ctx context.Context, file *adapter.File, source string) error {
	fileKey := fmt.Sprintf("%s:%s", source, file.Path)
	filenameKey := filepath.Base(file.Path) // Extract just the filename from the full path

	// Check if file already exists and hasn't changed
	// First check by full key (source:fullpath), then by filename only
	var existing *FileMetadata
	var exists bool

	if existing, exists = m.fileIndex[fileKey]; !exists {
		existing, exists = m.fileIndex[filenameKey]
		if exists {
			// Found by filename only, update to use full key
			delete(m.fileIndex, filenameKey)
			logrus.Debugf("Found existing file by filename, updating key from %s to %s", filenameKey, fileKey)
		}
	}

	if exists {
		// For files from OpenWebUI (source: "openwebui"), we can't compare content hashes
		// since OpenWebUI doesn't provide them. We'll assume they're unchanged if the file
		// exists and was found by filename matching.
		if existing.Source == "openwebui" {
			logrus.Debugf("File %s exists in OpenWebUI, skipping (no content hash available)", file.Path)
			// Update the file index to use the new key format
			m.fileIndex[fileKey] = existing
			return nil
		}

		// For files from adapters, compare content hashes
		if existing.Hash == file.Hash {
			logrus.Debugf("File %s unchanged, skipping", file.Path)
			return nil
		}
		logrus.Infof("File %s has changed, updating", file.Path)

		// Remove old file from knowledge if knowledge ID is set
		if m.knowledgeID != "" && existing.FileID != "" {
			logrus.Debugf("Removing old file %s from knowledge %s", existing.FileID, m.knowledgeID)
			if err := m.openwebuiClient.RemoveFileFromKnowledge(ctx, m.knowledgeID, existing.FileID); err != nil {
				logrus.Warnf("Failed to remove old file from knowledge: %v", err)
				// Continue with upload even if removal fails
			} else {
				logrus.Debugf("Successfully removed old file from knowledge")
			}
		}
	}

	// Save file to local storage
	localPath := filepath.Join(m.storagePath, "files", source, file.Path)
	if err := m.saveFileLocally(localPath, file.Content); err != nil {
		return fmt.Errorf("failed to save file locally: %w", err)
	}

	// Upload to OpenWebUI
	logrus.Debugf("Starting file upload to OpenWebUI for: %s", file.Path)
	uploadedFile, err := m.openwebuiClient.UploadFile(ctx, filepath.Base(file.Path), file.Content)
	if err != nil {
		return fmt.Errorf("failed to upload file to OpenWebUI: %w", err)
	}

	logrus.Debugf("File uploaded successfully: ID=%s, Filename=%s", uploadedFile.ID, uploadedFile.Filename)

	// Add to knowledge if knowledge ID is set
	if m.knowledgeID != "" {
		logrus.Debugf("Adding file %s to knowledge %s", uploadedFile.ID, m.knowledgeID)
		if err := m.openwebuiClient.AddFileToKnowledge(ctx, m.knowledgeID, uploadedFile.ID); err != nil {
			logrus.Errorf("Failed to add file to knowledge: %v", err)
			return fmt.Errorf("failed to add file to knowledge: %w", err)
		}
		logrus.Debugf("File successfully added to knowledge")
	} else {
		logrus.Warnf("No knowledge ID set, file uploaded but not added to any knowledge base")
	}

	// Update file index
	m.fileIndex[fileKey] = &FileMetadata{
		Path:     file.Path,
		Hash:     file.Hash,
		FileID:   uploadedFile.ID,
		Source:   source,
		SyncedAt: time.Now(),
		Modified: file.Modified,
	}

	logrus.Debugf("Updated file index with file: %s (ID: %s)", file.Path, uploadedFile.ID)
	logrus.Debugf("File index now contains %d files", len(m.fileIndex))

	logrus.Infof("Successfully synced file: %s", file.Path)
	return nil
}

// cleanupOrphanedFiles removes files from OpenWebUI that are no longer present in repositories
func (m *Manager) cleanupOrphanedFiles(ctx context.Context, currentFiles map[string]bool) error {
	logrus.Debugf("Checking for orphaned files...")

	var orphanedFiles []string
	for fileKey := range m.fileIndex {
		if !currentFiles[fileKey] {
			orphanedFiles = append(orphanedFiles, fileKey)
		}
	}

	if len(orphanedFiles) == 0 {
		logrus.Debugf("No orphaned files found")
		return nil
	}

	logrus.Infof("Found %d orphaned files to remove", len(orphanedFiles))

	for _, fileKey := range orphanedFiles {
		metadata := m.fileIndex[fileKey]

		// Remove from knowledge if knowledge ID is set
		if m.knowledgeID != "" && metadata.FileID != "" {
			logrus.Debugf("Removing orphaned file %s (ID: %s) from knowledge %s", metadata.Path, metadata.FileID, m.knowledgeID)
			if err := m.openwebuiClient.RemoveFileFromKnowledge(ctx, m.knowledgeID, metadata.FileID); err != nil {
				logrus.Warnf("Failed to remove orphaned file from knowledge: %v", err)
				// Continue with other files even if one fails
			} else {
				logrus.Debugf("Successfully removed orphaned file from knowledge")
			}
		}

		// Remove from file index
		delete(m.fileIndex, fileKey)
		logrus.Infof("Removed orphaned file: %s", metadata.Path)
	}

	return nil
}

// saveFileLocally saves a file to the local storage
func (m *Manager) saveFileLocally(path string, content []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// loadFileIndex loads the file index from disk
func (m *Manager) loadFileIndex() error {
	if _, err := os.Stat(m.indexPath); os.IsNotExist(err) {
		return nil // Index doesn't exist yet
	}

	data, err := os.ReadFile(m.indexPath)
	if err != nil {
		return fmt.Errorf("failed to read file index: %w", err)
	}

	if err := json.Unmarshal(data, &m.fileIndex); err != nil {
		return fmt.Errorf("failed to unmarshal file index: %w", err)
	}

	return nil
}

// saveFileIndex saves the file index to disk
func (m *Manager) saveFileIndex() error {
	logrus.Debugf("Saving file index to: %s", m.indexPath)
	logrus.Debugf("File index contains %d files", len(m.fileIndex))

	data, err := json.MarshalIndent(m.fileIndex, "", "  ")
	if err != nil {
		logrus.Errorf("Failed to marshal file index: %v", err)
		return fmt.Errorf("failed to marshal file index: %w", err)
	}

	logrus.Debugf("File index JSON size: %d bytes", len(data))

	if err := os.WriteFile(m.indexPath, data, 0644); err != nil {
		logrus.Errorf("Failed to write file index to %s: %v", m.indexPath, err)
		return fmt.Errorf("failed to write file index: %w", err)
	}

	logrus.Debugf("Successfully saved file index to: %s", m.indexPath)
	return nil
}

// GetFileHash calculates the hash of a file
func GetFileHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}
