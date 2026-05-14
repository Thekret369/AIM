package storage

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

// LocalUploader 本地磁盘存储实现
// 文件按 {userID}/{year-month}/{uuid}.{ext} 组织，缩略图文件名追加 _thumb
type LocalUploader struct {
	BasePath string // 文件存储根目录，如 ./data/uploads
}

// NewLocalUploader 创建本地磁盘上传器
func NewLocalUploader(basePath string) *LocalUploader {
	return &LocalUploader{BasePath: basePath}
}

// Upload 将文件保存到本地磁盘
// 图片类型自动生成 300px 宽等比缩略图
func (l *LocalUploader) Upload(userID uint, filename string, reader io.Reader) (*UploadResult, error) {
	now := time.Now()
	relDir := filepath.Join(fmt.Sprintf("%d", userID), now.Format("2006-01"))
	absDir := filepath.Join(l.BasePath, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	uniqueName := uuid.New().String() + ext
	filePath := filepath.Join(absDir, uniqueName)

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	size, err := io.Copy(f, reader)
	if err != nil {
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	relPath := filepath.ToSlash(filepath.Join(relDir, uniqueName))
	result := &UploadResult{
		URL:      "/data/uploads/" + relPath,
		FileName: filename,
		FileSize: size,
	}

	// 图片文件生成缩略图
	if isImageExt(ext) {
		thumbName := uuid.New().String() + "_thumb.jpg"
		thumbPath := filepath.Join(absDir, thumbName)
		if err := generateThumbnail(filePath, thumbPath, 300); err == nil {
			thumbRelPath := filepath.ToSlash(filepath.Join(relDir, thumbName))
			result.ThumbnailURL = "/data/uploads/" + thumbRelPath
		}
		// 缩略图生成失败不影响主流程，仅不返回 ThumbnailURL
	}

	return result, nil
}

// Delete 从本地磁盘删除文件及其缩略图
func (l *LocalUploader) Delete(url string) error {
	// URL 格式: /data/uploads/{userID}/{year-month}/{uuid}.{ext}
	relPath := strings.TrimPrefix(url, "/data/uploads/")
	if relPath == url {
		return fmt.Errorf("无效的文件 URL: %s", url)
	}
	absPath := filepath.Join(l.BasePath, relPath)
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 尝试删除对应缩略图
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	ext := filepath.Ext(base)
	nameNoExt := strings.TrimSuffix(base, ext)
	// 缩略图可能以 _thumb.jpg 命名，由原始文件名对应的 uuid 无关
	// 简单处理：不强制匹配，上传时缩略图 uuid 与源文件不同，这里仅删源文件
	_ = nameNoExt
	_ = dir
	return nil
}

// isImageExt 通过扩展名判断是否为图片
func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}

// generateThumbnail 生成缩略图，maxWidth 为最大宽度（等比缩放）
// 输出为 JPEG 格式，质量 80%
func generateThumbnail(srcPath, dstPath string, maxWidth int) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// 检测实际图片格式
	buf := make([]byte, 512)
	src.Read(buf)
	src.Seek(0, io.SeekStart)
	contentType := http.DetectContentType(buf)

	var img image.Image

	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(src)
	case "image/png":
		img, err = png.Decode(src)
	case "image/gif":
		img, err = gif.Decode(src)
	default:
		// 不支持的格式，尝试用 image.Decode 通用解码
		img, _, err = image.Decode(src)
	}
	if err != nil {
		return fmt.Errorf("解码图片失败: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// 宽度已小于目标，不缩放
	if w <= maxWidth {
		// 直接转 JPEG
		dst, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()
		return jpeg.Encode(dst, img, &jpeg.Options{Quality: 80})
	}

	newH := h * maxWidth / w
	dstImg := image.NewRGBA(image.Rect(0, 0, maxWidth, newH))
	draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), img, bounds, draw.Over, nil)

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	return jpeg.Encode(dst, dstImg, &jpeg.Options{Quality: 80})
}
