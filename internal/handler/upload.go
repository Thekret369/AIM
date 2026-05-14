package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"AIM/pkg/storage"

	"github.com/gin-gonic/gin"
)

// UploadHandler 文件上传处理器
type UploadHandler struct {
	Uploader storage.Uploader
}

// 允许的 MIME 类型前缀
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

// 禁止的文件扩展名
var blockedExts = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".sh": true,
	".ps1": true, ".vbs": true, ".com": true, ".msi": true,
}

// 文件大小上限：50MB
const maxUploadSize = 50 << 20

// HandleUpload 处理 multipart 文件上传
// POST /api/upload — 需 JWT 认证
func (h *UploadHandler) HandleUpload(c *gin.Context) {
	userID := c.GetUint("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到文件或文件过大（上限50MB）"})
		return
	}
	defer file.Close()

	// 校验扩展名黑名单
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if blockedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
		return
	}

	// 校验 MIME 类型
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

// isAllowedMime 校验 Content-Type 是否在允许范围内
func isAllowedMime(contentType string) bool {
	if contentType == "" {
		return true // 浏览器未提供时放行，依赖扩展名校验
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	for _, prefix := range allowedMimePrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}
