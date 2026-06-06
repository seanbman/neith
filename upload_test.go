package neith

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerUploadStoresMultipartFiles(t *testing.T) {
	previous := config
	tempDir := t.TempDir()
	config = &Config{
		Logger:          previous.Logger,
		UploadDir:       tempDir,
		UploadMaxBytes:  1 << 20,
		UploadMaxMemory: 1 << 20,
	}
	t.Cleanup(func() {
		config = previous
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/?neith_upload=1", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	newHandler().Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Files) != 1 {
		t.Fatalf("expected one uploaded file, got %d", len(response.Files))
	}

	upload := response.Files[0]
	if upload.FieldName != "avatar" {
		t.Fatalf("expected field avatar, got %q", upload.FieldName)
	}
	if upload.FileName != "hello.txt" {
		t.Fatalf("expected filename hello.txt, got %q", upload.FileName)
	}
	if upload.Size != 5 {
		t.Fatalf("expected size 5, got %d", upload.Size)
	}
	if filepath.Dir(upload.Path) != tempDir {
		t.Fatalf("expected upload in temp dir, got %q", upload.Path)
	}

	content, err := os.ReadFile(upload.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Fatalf("expected file content hello, got %q", string(content))
	}
}
