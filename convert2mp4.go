package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func convertWebmToMp4(inputPath, outputPath string) (err error) {
	cmd := exec.Command("ffmpeg", "-i", inputPath, outputPath)
	return cmd.Run()
}

func deleteOldUploadedFiles(uploadDir string) {
	// Get the current time
	now := time.Now()

	// Get a list of files in the upload directory
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		log.Fatalf("Failed to read upload directory: %v", err)
	}

	// Iterate over the files and delete the ones created more than 1 day ago
	for _, file := range files {
		if file.Type().IsRegular() {
			// Get the file info
			fileInfo, err := file.Info()
			if err != nil {
				log.Printf("Failed to get file info: %v", err)
				continue
			}

			// Calculate the age of the file
			age := now.Sub(fileInfo.ModTime())

			// Delete the file if it's older than 1 day
			if age > 24*time.Hour {
				err := os.Remove(filepath.Join(uploadDir, file.Name()))
				if err != nil {
					log.Printf("Failed to delete file: %v", err)
				} else {
					log.Printf("Deleted file: %s", file.Name())
				}
			}
		}
	}
}

func main() {
	uploadDir := "/data"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Call the deleteOldUploadedFiles routine periodically
	go func() {
		for {
			deleteOldUploadedFiles(uploadDir)
			time.Sleep(24 * time.Hour) // Run once every 24 hours
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()

			// Save the uploaded file to a temporary location
			tempFile, err := os.CreateTemp(uploadDir, "uploaded-file-*.webm")
			fullNameWithPath := tempFile.Name()
			if err != nil {
				log.Fatalf("Failed to create temporary file: %v", err)
			}
			defer tempFile.Close()

			// Copy the uploaded file to the temporary file
			_, err = io.Copy(tempFile, file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tempFile.Close()

			outputPath := fmt.Sprintf("%s/%v.mp4", uploadDir, time.Now().UnixNano())
			if err = convertWebmToMp4(fullNameWithPath, outputPath); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Serve the converted mp4 file for download
			http.ServeFile(w, r, outputPath)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
