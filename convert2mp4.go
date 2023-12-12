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

func deleteOldUploadedFiles(uploadDir string) {
	time.Sleep(time.Hour)
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
	go deleteOldUploadedFiles(uploadDir)
}

func main() {
	//uploadDir := "/data"
	uploadDir := os.Getenv("VideoCache")
	if uploadDir == "" {
		uploadDir = "/Users/yang/video"
	}
	log.Printf("\"VideoCache\" directory from environment variable: %s, current: %s", os.Getenv("VideoCache"), uploadDir)

	//err if the directory does not exist
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		//try to create the directory
		err = os.Mkdir(uploadDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create upload directory: %v", err)
			return
		}
		log.Printf("Created upload directory: %s", uploadDir)
	} else {
		log.Printf("Upload directory existed: %s", uploadDir)
	}

	// Call the deleteOldUploadedFiles routine periodically
	go deleteOldUploadedFiles(uploadDir)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var (
			mp4       []byte
			paramName string
		)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if paramName = r.URL.Query().Get("name"); paramName == "" {
			log.Printf("parameter name is required but missing")
			http.Error(w, "parameter name is required but missing", http.StatusBadRequest)
			return
		}

		// Save the uploaded file to a temporary location
		tempFile, err := os.CreateTemp(uploadDir, fmt.Sprintf("uploaded-file-%s.webm", paramName))
		inputPath := tempFile.Name()
		if err != nil {
			log.Fatalf("Failed to create temporary file: %v", err)
		}
		defer tempFile.Close()
		defer os.Remove(inputPath)

		// Copy the posted to the temporary file
		_, err = io.Copy(tempFile, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tempFile.Close()

		outputPath := fmt.Sprintf("%s/mp4-%s.mp4", uploadDir, paramName)
		//ffmpeg -i /Users/yang/video/uploaded-file-退休规划\:确保金融安全的退休生 活.webm2812214445  -c:v libx265 -c:v hevc_videotoolbox /Users/yang/video/o.mp4
		//cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "libx265", "-c:v", "hevc_videotoolbox", outputPath)
		cmd := exec.Command("ffmpeg", "-i", inputPath, outputPath)
		if err = cmd.Run(); err != nil {
			log.Printf("Failed to convert video: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(outputPath)

		// Serve the converted mp4 file for download
		//write back the mp4 file converted
		if mp4, err = os.ReadFile(outputPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if len(mp4) == 0 {
			http.Error(w, "Empty mp4 file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", "attachment; filename=video.mp4")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mp4)))
		if _, err = w.Write(mp4); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	//open the port 8001
	log.Printf("Listening on port 8001")
	log.Fatal(http.ListenAndServe(":8001", nil))

}
