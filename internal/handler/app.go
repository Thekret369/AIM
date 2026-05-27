package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type AppHandler struct{}

type androidUpdateResponse struct {
	HasUpdate         bool     `json:"has_update"`
	LatestVersionCode int      `json:"latest_version_code"`
	LatestVersionName string   `json:"latest_version_name"`
	DownloadURL       string   `json:"download_url,omitempty"`
	ChecksumSHA256    string   `json:"checksum_sha256,omitempty"`
	Requirement       string   `json:"requirement"`
	ReleaseNotes      []string `json:"release_notes"`
}

// GetAndroidLatest returns the current Android package metadata.
// Values can be injected during deployment without changing code.
func (h *AppHandler) GetAndroidLatest(c *gin.Context) {
	currentCode, _ := strconv.Atoi(c.Query("version_code"))
	latestCode := envInt("LANLINE_ANDROID_VERSION_CODE", 7)
	latestName := envString("LANLINE_ANDROID_VERSION_NAME", "0.7.0-debug")
	requirement := envString("LANLINE_ANDROID_UPDATE_REQUIREMENT", "Optional")

	c.JSON(http.StatusOK, androidUpdateResponse{
		HasUpdate:         currentCode > 0 && latestCode > currentCode,
		LatestVersionCode: latestCode,
		LatestVersionName: latestName,
		DownloadURL:       strings.TrimSpace(os.Getenv("LANLINE_ANDROID_APK_URL")),
		ChecksumSHA256:    strings.TrimSpace(os.Getenv("LANLINE_ANDROID_APK_SHA256")),
		Requirement:       normalizeUpdateRequirement(requirement),
		ReleaseNotes:      splitReleaseNotes(os.Getenv("LANLINE_ANDROID_RELEASE_NOTES")),
	})
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func normalizeUpdateRequirement(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "required", "force", "forced":
		return "Required"
	case "recommended":
		return "Recommended"
	default:
		return "Optional"
	}
}

func splitReleaseNotes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "|", "\n")
	parts := strings.Split(raw, "\n")
	notes := make([]string, 0, len(parts))
	for _, part := range parts {
		if note := strings.TrimSpace(part); note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}
