package handlers

import (
	"clipper/internal/models"
	"os"
	"os/exec"
	"sync"
)

type sharedData struct {

	// fileIDs maps a unique process ID to the full path of the downloaded file.
	fileIDs map[string]string

	// progressTracker maps process IDs to their progress channels
	progressTracker map[string]chan models.ProgressResponse

	// processes maps process IDs to their ffmpeg *exec.Cmd
	// it is used to stop the ffmpeg process if the client disconnects
	processes map[string]*exec.Cmd

	// mu protects concurrent access to fileIDs, progressTracker, and processes.
	mu sync.RWMutex
}

func (s *sharedData) addFileID(fileID, filePath string) {
	s.mu.Lock()
	s.fileIDs[fileID] = filePath
	s.mu.Unlock()
}

func (s *sharedData) getFilePath(fileID string) (string, bool) {
	s.mu.RLock()
	filePath, exists := s.fileIDs[fileID]
	s.mu.RUnlock()
	return filePath, exists
}

func (s *sharedData) addProgressChannel(fileID string, channel chan models.ProgressResponse) {
	s.mu.Lock()
	s.progressTracker[fileID] = channel
	s.mu.Unlock()
}

func (s *sharedData) getProgressChannel(fileID string) (chan models.ProgressResponse, bool) {
	s.mu.RLock()
	channel, exists := s.progressTracker[fileID]
	s.mu.RUnlock()
	return channel, exists
}

func (s *sharedData) removeProgressChannel(fileID string) {
	s.mu.Lock()
	delete(s.progressTracker, fileID)
	s.mu.Unlock()
}

func (s *sharedData) removeFileID(fileID string) {
	s.mu.Lock()
	delete(s.fileIDs, fileID)
	s.mu.Unlock()
}

func (s *sharedData) addProcess(fileID string, cmd *exec.Cmd) {
	s.mu.Lock()
	s.processes[fileID] = cmd
	s.mu.Unlock()
}

func (s *sharedData) getProcess(fileID string) (*exec.Cmd, bool) {
	s.mu.RLock()
	cmd, exists := s.processes[fileID]
	s.mu.RUnlock()
	return cmd, exists
}

func (s *sharedData) removeProcess(fileID string) {
	s.mu.Lock()
	delete(s.processes, fileID)
	s.mu.Unlock()
}

func (s *sharedData) cleanupAll(fileID string) {
	filePath, _ := s.getFilePath(fileID)
	os.Remove(filePath)
	s.removeFileID(fileID)
	s.removeProgressChannel(fileID)
	s.removeProcess(fileID)

}

var data = &sharedData{
	fileIDs:         make(map[string]string),
	progressTracker: make(map[string]chan models.ProgressResponse),
	processes:       make(map[string]*exec.Cmd),
	mu:              sync.RWMutex{},
}
