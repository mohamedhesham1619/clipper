package handlers

import (
	"clipper/internal/models"
	"sync"
)

// fileIDs maps a unique process ID to the full path of the downloaded file.
var fileIDs = make(map[string]string)

// progressTracker maps process IDs to their progress channels
var progressTracker = make(map[string]chan models.ProgressResponse)

// mu protects concurrent access to fileIDs and progressTracker.
var mu sync.RWMutex
