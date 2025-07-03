package handlers

import (
	"clipper/internal/models"
	"sync"
)

// fileIDs maps a unique process ID to the full path of the downloaded file.
var fileIDs = make(map[string]string)

// progressTracker maps process IDs to their progress channels
var progressTracker = make(map[string]chan models.ProgressResponse)

// jobStatus tracks the final state of a job (completed, failed)
var jobStatus = make(map[string]string)

// mu protects concurrent access to fileIDs, progressTracker, and jobStatus
var mu sync.RWMutex
