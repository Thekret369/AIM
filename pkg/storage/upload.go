// Package storage 提供文件（图片、文件、音频）的上传与存储能力
// 当前占位，后续实现本地存储 / MinIO / OSS 适配
package storage

import "io"

// Uploader 文件上传接口
type Uploader interface {
	Upload(filename string, reader io.Reader) (url string, err error)
	Delete(url string) error
}
