package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAppHandlerGetAndroidLatestDefaults(t *testing.T) {
	response := requestAndroidLatest(t, "/api/app/android/latest?version_code=6")

	if response.HasUpdate {
		t.Fatal("expected current default version to be latest")
	}
	if response.LatestVersionCode != 6 || response.LatestVersionName != "0.6.0-debug" {
		t.Fatalf("unexpected default version: %+v", response)
	}
	if response.Requirement != "Optional" {
		t.Fatalf("unexpected default requirement: %s", response.Requirement)
	}
}

func TestAppHandlerGetAndroidLatestFromEnvironment(t *testing.T) {
	t.Setenv("LANLINE_ANDROID_VERSION_CODE", "8")
	t.Setenv("LANLINE_ANDROID_VERSION_NAME", "0.8.0")
	t.Setenv("LANLINE_ANDROID_APK_URL", "https://download.example/lanline.apk")
	t.Setenv("LANLINE_ANDROID_APK_SHA256", "abc123")
	t.Setenv("LANLINE_ANDROID_UPDATE_REQUIREMENT", "required")
	t.Setenv("LANLINE_ANDROID_RELEASE_NOTES", "后端推送|本地通知")

	response := requestAndroidLatest(t, "/api/app/android/latest?version_code=4")

	if !response.HasUpdate {
		t.Fatal("expected older client to receive update flag")
	}
	if response.LatestVersionCode != 8 || response.LatestVersionName != "0.8.0" {
		t.Fatalf("unexpected env version: %+v", response)
	}
	if response.DownloadURL == "" || response.ChecksumSHA256 != "abc123" {
		t.Fatalf("unexpected package metadata: %+v", response)
	}
	if response.Requirement != "Required" {
		t.Fatalf("unexpected requirement: %s", response.Requirement)
	}
	if len(response.ReleaseNotes) != 2 {
		t.Fatalf("unexpected release notes: %+v", response.ReleaseNotes)
	}
}

func requestAndroidLatest(t *testing.T, target string) androidUpdateResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &AppHandler{}
	router.GET("/api/app/android/latest", handler.GetAndroidLatest)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var response androidUpdateResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}
