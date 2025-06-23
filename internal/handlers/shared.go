package handlers

import "clipper/internal/models"

// fileIDs maps file names to their unique IDs
var fileIDs = make(map[string]string)

// progressTracker maps file IDs to their progress channels
var progressTracker = make(map[string]chan models.ProgressResponse)
