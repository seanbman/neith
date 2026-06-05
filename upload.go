package fcmp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type uploadResponse struct {
	Files []Upload `json:"files"`
}

func (h handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxBytes := config.UploadMaxBytes
	if maxBytes <= 0 {
		maxBytes = 64 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	maxMemory := config.UploadMaxMemory
	if maxMemory <= 0 {
		maxMemory = 32 << 20
	}
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		http.Error(w, "invalid multipart upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	files, err := saveUploadedFiles(r)
	if err != nil {
		http.Error(w, "failed to save upload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(uploadResponse{Files: files})
}

func saveUploadedFiles(r *http.Request) ([]Upload, error) {
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		return nil, nil
	}

	dir := uploadDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	var uploads []Upload
	for fieldName, headers := range r.MultipartForm.File {
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return nil, err
			}

			uploadID := uuid.New().String()
			fileName := filepath.Base(header.Filename)
			path := filepath.Join(dir, fmt.Sprintf("%s-%s", uploadID, fileName))

			dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				_ = file.Close()
				return nil, err
			}

			size, copyErr := io.Copy(dst, file)
			closeErr := dst.Close()
			_ = file.Close()
			if copyErr != nil {
				return nil, copyErr
			}
			if closeErr != nil {
				return nil, closeErr
			}

			uploads = append(uploads, Upload{
				ID:          uploadID,
				FieldName:   fieldName,
				FileName:    fileName,
				ContentType: header.Header.Get("Content-Type"),
				Size:        size,
				Path:        path,
			})
		}
	}
	return uploads, nil
}

func uploadDir() string {
	if config.UploadDir != "" {
		return config.UploadDir
	}
	return filepath.Join(os.TempDir(), "fcmp-uploads")
}
