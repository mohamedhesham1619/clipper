package models

type ProgressResponse struct{
	Status string `json:"status"`
	Progress int	`json:"progress"`
	DownloadUrl string `json:"downloadUrl"`
}