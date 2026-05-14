// Package storage 提供文件（图片、文件、音频）的上传与存储能力
package storage

import "io"

// UploadResult 上传结果
type UploadResult struct {
	URL          string `json:"url"`                     // 文件访问 URL
	ThumbnailURL string `json:"thumbnail_url,omitempty"` // 缩略图 URL（仅图片）
	FileName     string `json:"file_name"`               // 原始文件名
	FileSize     int64  `json:"file_size"`               // 文件大小(字节)
}

// Uploader 文件上传接口
// 上传文件到存储后端，返回可访问的 URL
type Uploader interface {
	Upload(userID uint, filename string, reader io.Reader) (*UploadResult, error)
	Delete(url string) error
}
