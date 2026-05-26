package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"LanLine/pkg/storage"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	Uploader storage.Uploader
}

var allowedMimePrefixes = []string{
	"image/",
	"audio/",
	"video/",
	"application/pdf",
	"application/msword",
	"application/vnd.openxmlformats",
	"application/vnd.ms-",
	"application/zip",
	"application/x-rar",
	"application/x-7z",
	"application/octet-stream",
	"text/",
}

var blockedExts = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".sh": true,
	".ps1": true, ".vbs": true, ".com": true, ".msi": true,
}

const maxUploadSize = 50 << 20

func (h *UploadHandler) HandleUpload(c *gin.Context) {
	userID := c.GetUint("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到文件或文件过大（上限50MB）"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if blockedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !isAllowedMime(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型: " + contentType})
		return
	}

	result, err := h.Uploader.Upload(userID, header.Filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":           result.URL,
		"thumbnail_url": result.ThumbnailURL,
		"file_name":     result.FileName,
		"file_size":     result.FileSize,
	})
}

func isAllowedMime(contentType string) bool {
	if contentType == "" {
		return true
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	for _, prefix := range allowedMimePrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}
