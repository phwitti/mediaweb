package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// For testing purposes
var ffmpegCmd = "ffmpeg"

// videoThumbnailSupport returns true if ffmpeg is installed, and thus
// video thumbnails is supported
func hasVideoThumbnailSupport() bool {
	_, err := exec.LookPath(ffmpegCmd)
	return err == nil
}

// getFiles returns a slice of File's sorted on file name
func getFiles(fullPath string, relativePath string) ([]File, error) {
	files := make([]File, 0, 500)
	fileInfos, err := os.ReadDir(fullPath)
	if err != nil {
		return files, err
	}

	for _, dirEntry := range fileInfos {
		fileInfo, _ := dirEntry.Info()
		fileType := ""
		if dirEntry.IsDir() || fileInfo.Mode()&os.ModeSymlink != 0 {
			fileType = "folder"
		} else {
			fileType = getFileType(dirEntry.Name())
		}
		// Only add directories, videos and images
		if fileType != "" {
			// Use path with / slash
			pathOriginal := filepath.Join(relativePath, dirEntry.Name())
			pathNew := filepath.ToSlash(pathOriginal)

			file := File{
				Type: fileType,
				Name: dirEntry.Name(),
				Path: pathNew}
			files = append(files, file)
		} else {
			log.Debug("getFiles - omitting:", fileInfo.Name())
		}
	}
	return files, nil
}

// getFileType returns "video" for video files and "image" for image files.
// For all other files (including folders) "" is returned.
// relativeFileName can also include an absolute or relative path.
func getFileType(relativeFileName string) string {

	// Check if this is an image
	if isImage(relativeFileName) {
		return "image"
	}

	// Check if this is a video
	if isVideo(relativeFileName) {
		return "video"
	}

	return "" // Not a video nor an image
}

func isImage(pathAndFile string) bool {
	extension := filepath.Ext(pathAndFile)
	for _, imgExtension := range imgExtensions {
		if strings.EqualFold(extension, imgExtension) {
			return true
		}
	}
	return false
}

func isVideo(pathAndFile string) bool {
	extension := filepath.Ext(pathAndFile)
	for _, vidExtension := range vidExtensions {
		if strings.EqualFold(extension, vidExtension) {
			return true
		}
	}
	return false
}

// contains is a helper function to find a string within
// a slice of multiple strings
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
