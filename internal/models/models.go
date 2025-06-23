package models

type ProgressResponse struct{
	Status string `json:"status"`
	Progress int	`json:"progress"`
	DownloadUrl string `json:"downloadUrl"`
}

type VideoRequest struct {
	VideoURL     string `json:"videoUrl"`
	ClipStart string `json:"clipStart"`
	ClipEnd string `json:"clipEnd"`
	Quality string `json:"quality"`
}