package main

import (
	"bytes"
	"fmt"
	"hash"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	log "github.com/sirupsen/logrus"
)

// Cache keeps information about all known cache items
type Cache struct {
	cachepath                string // Top level path for thumbnails and previews
	previewMaxSide           int
	genPreviewForSmallImages bool
	genAlbumThumbs           bool
	thumbnails               map[string]time.Time // Key: relativePath of thumbnail to cachepath, Value: time of last update
	previews                 map[string]time.Time // Key: relativePath of preview to cachepath, Value: time of last update
	albumThumbnails          map[string]time.Time // Key: relativePath of preview to cachepath, Value: time of last update
}

func createCache(m *Media, cachepath string, previewMaxSide int, genPreviewForSmallImages bool, genAlbumThumbs bool) *Cache {
	c := &Cache{
		cachepath:                cachepath,
		previewMaxSide:           previewMaxSide,
		genPreviewForSmallImages: genPreviewForSmallImages,
		genAlbumThumbs:           genAlbumThumbs,
		thumbnails:               map[string]time.Time{},
		previews:                 map[string]time.Time{}}
	c.loadCache("", true)
	return c
}

func (c *Cache) loadCache(relativePath string, recursive bool) {
	fullCachePath, err := c.getFullCachePath(relativePath)
	if err != nil {
		return
	}

	cacheFiles, err := getFiles(fullCachePath, relativePath)
	if err != nil {
		return
	}

	for _, file := range cacheFiles {
		if file.Type == "folder" {
			if recursive {
				c.loadCache(file.Path, true) // Recursive
			}
		} else if file.Type == "image" {
			if strings.HasSuffix(file.Name, ".preview.jpg") {
				c.previews[file.Path] = time.Now()
			} else if strings.HasSuffix(file.Name, ".thumb.jpg") {
				c.thumbnails[file.Path] = time.Now()
			}
		}
	}
}

func (c *Cache) hasThumbnail(relativeMediaPath string) bool {
	path, err := c.relativeThumbnailPath(relativeMediaPath)
	if err != nil {
		log.Warn(err)
		return false
	}
	_, ok := c.thumbnails[path]
	return ok
}

func (c *Cache) hasPreview(relativeMediaPath string) bool {
	path, err := c.relativePreviewPath(relativeMediaPath)
	if err != nil {
		log.Warn(err)
		return false
	}
	_, ok := c.previews[path]
	return ok
}

func (c *Cache) hasAlbumThumbnail(relativeAlbumPreviewPath string) bool {
	_, ok := c.albumThumbnails[relativeAlbumPreviewPath]
	return ok
}

// getFullCachePath returns the full path of the provided path, i.e:
// thumb path + relative path.
func (c *Cache) getFullCachePath(relativePath string) (string, error) {
	return getFullPath(c.cachepath, relativePath)
}

// thumbnailPath returns the absolute thumbnail file path from a
// media path. Thumbnails are always stored in JPEG format (.jpg
// extension) and starts with '_'.
// Returns error if the media path is invalid.
func (c *Cache) thumbnailPath(relativeMediaPath string) (string, error) {
	relativePath, err := c.relativeThumbnailPath(relativeMediaPath)
	if err != nil {
		return "", err
	}
	return c.getFullCachePath(relativePath)
}

func (c *Cache) relativeThumbnailPath(relativeMediaPath string) (string, error) {
	path, file := filepath.Split(relativeMediaPath)
	// Replace extension with .thumb.jpg
	ext := filepath.Ext(file)
	if ext == "" {
		return "", fmt.Errorf("File has no extension: %s", file)
	}
	file = strings.Replace(file, ext, ".thumb.jpg", -1)
	return filepath.ToSlash(filepath.Join(path, file)), nil
}

// previewPath returns the absolute preview file path from a
// media path. Previews are always stored in JPEG format (.jpg
// extension) and starts with 'view_'.
// Returns error if the media path is invalid.
func (c *Cache) previewPath(relativeMediaPath string) (string, error) {
	relativePath, err := c.relativePreviewPath(relativeMediaPath)
	if err != nil {
		return "", err
	}
	return c.getFullCachePath(relativePath)
}

func (c *Cache) relativePreviewPath(relativeMediaPath string) (string, error) {
	path, file := filepath.Split(relativeMediaPath)
	// Replace extension with .preview.jpg
	ext := filepath.Ext(file)
	if ext == "" {
		return "", fmt.Errorf("file has no extension: %s", file)
	}
	file = strings.Replace(file, ext, ".preview.jpg", -1)
	return filepath.ToSlash(filepath.Join(path, file)), nil
}

var fnvHash hash.Hash64

func (c *Cache) relativeAlbumThumbnailPath(relativeAlbumPath string, files []string) string {
	if fnvHash == nil {
		fnvHash = fnv.New64()
	}

	_, folder := filepath.Split(relativeAlbumPath)

	fnvHash.Reset()
	fnvHash.Write([]byte(folder + strings.Join(files, "")))
	return filepath.Join(relativeAlbumPath, strconv.FormatUint(fnvHash.Sum64(), 10)) + ".jpg"
}

// errorIndicationPath returns the file path with the extension
// replaced with err.
func (c *Cache) errorIndicationPath(anyPath string) string {
	path, file := filepath.Split(anyPath)
	ext := filepath.Ext(file)
	file = strings.Replace(file, ext, ".err.txt", -1)
	return filepath.Join(path, file)
}

// generateTumbnail generates a thumbnail for an image or video
// and returns the file name of the thumbnail. If a thumbnail already
// exist the file name will be returned.
func (c *Cache) generateThumbnail(m *Media, relativeFilePath string) (string, error) {
	relativeThumbPath, err := c.relativeThumbnailPath(relativeFilePath)
	if err != nil {
		log.Warn(err)
		return "", err
	}
	thumbFileName, err := c.getFullCachePath(relativeThumbPath)
	if err != nil {
		log.Warn(err)
		return "", err
	}
	_, err = os.Stat(thumbFileName) // Check if file exist
	if err == nil {
		return thumbFileName, nil // Thumb already generated
	}
	errorIndicationFile := c.errorIndicationPath(thumbFileName)
	_, err = os.Stat(errorIndicationFile) // Check if file exist
	if err == nil {
		// File has failed to be generated before, don't bother
		// trying to re-generate it.
		msg := fmt.Sprintf("skipping generate thumbnail for %s since it has failed before,", relativeFilePath)
		log.Trace(msg)
		return "", fmt.Errorf(msg)
	}

	// No thumb exist. Create it
	log.Info("Creating new thumbnail for ", relativeFilePath)
	startTime := time.Now().UnixNano()
	fullMediaPath, err := m.getFullMediaPath(relativeFilePath)
	if err != nil {
		log.Warn(err)
		return "", err
	}
	if isVideo(fullMediaPath) {
		err = c.generateVideoThumbnail(fullMediaPath, thumbFileName)
	} else {
		err = c.generateImageThumbnail(fullMediaPath, thumbFileName)
	}
	if err != nil {
		// To avoid generate the file again, create an error indication file
		c.generateErrorIndicationFile(errorIndicationFile, err)
		return "", err
	}

	c.thumbnails[relativeThumbPath] = time.Now()

	deltaTime := (time.Now().UnixNano() - startTime) / int64(time.Millisecond)
	log.Infof("Thumbnail done for %s (conversion time: %d ms)", relativeFilePath, deltaTime)
	return thumbFileName, nil
}

// generatePreview generates a preview image and returns the file name of the
// preview. If a preview file already exist the file name will be returned.
func (c *Cache) generatePreview(m *Media, relativeFilePath string) (string, bool, error) {
	relativePreviewPath, err := c.relativePreviewPath(relativeFilePath)
	if err != nil {
		log.Warn(err)
		return "", false, err
	}
	previewFileName, err := c.getFullCachePath(relativePreviewPath)
	if err != nil {
		log.Warn(err)
		return "", false, err
	}
	_, err = os.Stat(previewFileName) // Check if file exist
	if err == nil {
		return previewFileName, false, nil // Preview already generated
	}

	errorIndicationFile := c.errorIndicationPath(previewFileName)
	_, err = os.Stat(errorIndicationFile) // Check if file exist
	if err == nil {
		// File has failed to be generated before, don't bother
		// trying to re-generate it.
		msg := fmt.Sprintf("Skipping generate preview for %s since it has failed before.",
			relativeFilePath)
		log.Trace(msg)
		return "", false, fmt.Errorf(msg)
	}

	fullMediaPath, err := m.getFullMediaPath(relativeFilePath)
	if err != nil {
		log.Warn(err)
		return "", false, err
	}

	width, height, err := m.getImageWidthAndHeight(fullMediaPath)
	if err != nil {
		// To avoid generate the file again, create an error indication file
		c.generateErrorIndicationFile(errorIndicationFile, err)
		return "", false, err
	}

	if !c.genPreviewForSmallImages && width <= c.previewMaxSide && height <= c.previewMaxSide {
		msg := fmt.Sprintf("Image %s too small to generate preview", relativeFilePath)
		log.Trace(msg)
		return "", true, fmt.Errorf(msg)
	}

	// No preview exist. Create it
	log.Info("Creating new preview file for ", relativeFilePath)
	startTime := time.Now().UnixNano()
	err = c.generateImagePreview(fullMediaPath, previewFileName)
	if err != nil {
		// To avoid generate the file again, create an error indication file
		c.generateErrorIndicationFile(errorIndicationFile, err)
		return "", false, err
	}

	c.previews[relativePreviewPath] = time.Now()

	deltaTime := (time.Now().UnixNano() - startTime) / int64(time.Millisecond)
	log.Infof("Preview done for %s (conversion time: %d ms)", relativeFilePath, deltaTime)
	return previewFileName, false, nil
}

func (c *Cache) generateAlbumThumbnail(m *Media, relativeAlbumPreviewPath string, albumPath string, files []string) error {
	relativePreviewPath, err := c.relativePreviewPath(relativeAlbumPreviewPath)
	if err != nil {
		log.Warn(err)
		return err
	}
	previewFileName, err := c.getFullCachePath(relativePreviewPath)
	if err != nil {
		log.Warn(err)
		return err
	}
	_, err = os.Stat(previewFileName) // Check if file exist
	if err == nil {
		return nil // Preview already generated
	}

	var thumbImg image.Image
	if len(files) <= 4 {
		thumbImg = c.generateAlbumThumbnail_4x4(m, albumPath, files)
	} else {
		thumbImg = c.generateAlbumThumbnail_9x9(m, albumPath, files)
	}

	// Create subdirectories if needed
	directory := filepath.Dir(previewFileName)
	err = os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directories in %s for creating thumbnail, reason %s", previewFileName, err)
	}

	// Write thumbnail to file
	outFile, err := os.Create(previewFileName)
	if err != nil {
		return fmt.Errorf("unable to open %s for creating thumbnail, reason %s", previewFileName, err)
	}
	defer outFile.Close()
	err = imaging.Encode(outFile, thumbImg, imaging.JPEG)

	return err
}

func (c *Cache) generateAlbumThumbnail_4x4(m *Media, albumPath string, files []string) image.Image {
	size_org := 256
	size_small := 128
	positionsX := []int{0, 1, 0, 1}
	positionsY := []int{0, 0, 1, 1}

	var err error
	var thumbImg image.Image
	thumbImg = imaging.New(size_org, size_org, color.Black)

	index := 0
	for _, file := range files {
		thumbImg, err = c.placeImageThumbnail(m, thumbImg, filepath.Join(albumPath, file), size_small, positionsX[index], positionsY[index])
		if err != nil {
			continue
		}

		index = index + 1
		if index >= 4 {
			break
		}
	}
	return thumbImg
}

func (c *Cache) generateAlbumThumbnail_9x9(m *Media, albumPath string, files []string) image.Image {
	size_org := 256
	size_small := 86
	positionsX := []int{0, 1, 2, 0, 1, 2, 0, 1, 2}
	positionsY := []int{0, 0, 0, 1, 1, 1, 2, 2, 2}

	var err error
	var thumbImg image.Image
	thumbImg = imaging.New(size_org, size_org, color.Black)

	index := 0
	for _, file := range files {
		thumbImg, err = c.placeImageThumbnail(m, thumbImg, filepath.Join(albumPath, file), size_small, positionsX[index], positionsY[index])
		if err != nil {
			continue
		}

		index = index + 1
		if index >= 9 {
			break
		}
	}
	return thumbImg
}

func (c *Cache) placeImageThumbnail(m *Media, thumb image.Image, relativeMediaPath string, size int, positionX int, positionY int) (image.Image, error) {
	var buffer bytes.Buffer
	err := m.writeThumbnail(&buffer, relativeMediaPath)
	if err != nil {
		return nil, err
	}
	img, err := imaging.Decode(&buffer)
	if err != nil {
		return nil, err
	}
	smallImg := imaging.Resize(img, size, size, imaging.Box)
	return imaging.Paste(thumb, smallImg, image.Point{X: positionX * size, Y: positionY * size}), nil
}

// generateErrorIndication creates a text file including the error reason.
func (c *Cache) generateErrorIndicationFile(errorIndicationFile string, err error) {
	log.Warn(err)
	errorFile, err2 := os.Create(errorIndicationFile)
	if err2 == nil {
		defer errorFile.Close()
		errorFile.WriteString(err.Error())
		log.Info("Created: ", errorIndicationFile)
	} else {
		log.Warnf("Unable to create %s. Reason: %s", errorIndicationFile, err2)
	}
}

// generateImageThumbnail generates a thumbnail from any of the supported
// images. Will create necessary subdirectories in the thumbpath.
func (c *Cache) generateImageThumbnail(fullMediaPath, fullThumbPath string) error {
	img, err := imaging.Open(fullMediaPath, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("unable to open image %s, reason: %s", fullMediaPath, err)
	}
	thumbImg := imaging.Thumbnail(img, 256, 256, imaging.Box)

	// Create subdirectories if needed
	directory := filepath.Dir(fullThumbPath)
	err = os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directories in %s for creating thumbnail, reason %s", fullThumbPath, err)
	}

	// Write thumbnail to file
	outFile, err := os.Create(fullThumbPath)
	if err != nil {
		return fmt.Errorf("unable to open %s for creating thumbnail, reason %s", fullThumbPath, err)
	}
	defer outFile.Close()
	err = imaging.Encode(outFile, thumbImg, imaging.JPEG)

	return err
}

// generateImagePreview generates a preview from any of the supported
// images. Will create necessary subdirectories in the PreviewPath.
func (c *Cache) generateImagePreview(fullMediaPath, fullPreviewPath string) error {
	img, err := imaging.Open(fullMediaPath, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("unable to open image %s, reason: %s", fullMediaPath, err)
	}
	previewImg := imaging.Fit(img, c.previewMaxSide, c.previewMaxSide, imaging.Box)

	// Create subdirectories if needed
	directory := filepath.Dir(fullPreviewPath)
	err = os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directories in %s for creating preview, reason %s", fullPreviewPath, err)
	}

	// Write thumbnail to file
	outFile, err := os.Create(fullPreviewPath)
	if err != nil {
		return fmt.Errorf("unable to open %s for creating preview, reason %s", fullPreviewPath, err)
	}
	defer outFile.Close()
	err = imaging.Encode(outFile, previewImg, imaging.JPEG)

	return err
}

// generateVideoThumbnail generates a thumbnail from any of the supported
// videos. Will create necessary subdirectories in the thumbpath.
func (c *Cache) generateVideoThumbnail(fullMediaPath, fullThumbPath string) error {
	// The temporary file for the screenshot
	screenShot := fullThumbPath + ".sh.jpg"

	// Extract the screenshot
	err := c.extractVideoScreenshot(fullMediaPath, screenShot)
	if err != nil {
		return err
	}
	defer os.Remove(screenShot) // Remove temporary file

	// Generate thumbnail from the screenshot
	img, err := imaging.Open(screenShot, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("unable to open screenshot image %s, reason: %s", screenShot, err)
	}
	thumbImg := imaging.Thumbnail(img, 256, 256, imaging.Box)

	// Add small video icon i upper right corner to indicate that this is
	// a video
	iconVideoImg, err := c.getVideoIcon()
	if err != nil {
		return err
	}
	thumbImg = imaging.Overlay(thumbImg, iconVideoImg, image.Pt(155, 11), 1.0)

	// Write thumbnail to file
	outFile, err := os.Create(fullThumbPath)
	if err != nil {
		return fmt.Errorf("unable to open %s for creating thumbnail, reason %s", fullThumbPath, err)
	}
	defer outFile.Close()
	err = imaging.Encode(outFile, thumbImg, imaging.JPEG)

	return err
}

// extractVideoScreenshot extracts a screenshot from a video using external
// ffmpeg software. Will create necessary directories in the outFilePath
func (c *Cache) extractVideoScreenshot(inFilePath, outFilePath string) error {
	if !hasVideoThumbnailSupport() {
		return fmt.Errorf("video thumbnails not supported. ffmpeg not installed")
	}

	// Create subdirectories if needed
	directory := filepath.Dir(outFilePath)
	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directories in %s for extracting screenshot, reason %s", outFilePath, err)
	}

	// Define argments for ffmpeg
	ffmpegArgs := []string{
		"-i",
		inFilePath,
		"-ss",
		"00:00:05", // 5 seconds into movie
		"-vframes",
		"1",
		outFilePath}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	//cmd := exec.Command(ffmpegCmd, ffmpegArg)
	cmd := exec.Command(ffmpegCmd, ffmpegArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	_, outFileErr := os.Stat(outFilePath)
	if err != nil || outFileErr != nil {
		return fmt.Errorf("%s %s\nStdout: %s\nStderr: %s",
			ffmpegCmd, strings.Join(ffmpegArgs, " "), stdout.String(), stderr.String())
	}
	return nil
}

// Cache to avoid regenerate icon each time (do it once)
var videoIcon image.Image

func (c *Cache) getVideoIcon() (image.Image, error) {
	if videoIcon != nil {
		// To avoid re-generate
		return videoIcon, nil
	}
	var err error
	videoIcon, err = imaging.Decode(bytes.NewReader(embedVideoIconBytes))
	if err != nil {
		return nil, err
	}
	videoIcon = imaging.Resize(videoIcon, 90, 90, imaging.Box)
	return videoIcon, nil
}

// cleanupCache removes all files and directories in the cache directory
// which don't have any corresponding media file.
// relativePath relative path where to clean up cache files.
// expectedMediaFiles are all files, including directories that are allowed
// as thumbs, preview or error files in the cache.
// Returns number of removed files and directories
func (c *Cache) cleanupCache(relativePath string, expectedMediaFiles []File) int {
	fullCachePath, _ := c.getFullCachePath(relativePath)
	log.Debug("Cleaning up directory: ", fullCachePath)

	// Figure possible directories, thumb, preview and error file names
	cacheFileNames := make([]string, 0, len(expectedMediaFiles)*5)
	for _, file := range expectedMediaFiles {
		_, fileName := filepath.Split(file.Name)
		if file.Type == "folder" {
			cacheFileNames = append(cacheFileNames, fileName)
		} else {
			thumbName, err := c.thumbnailPath(fileName)
			if err == nil {
				_, thumbName = filepath.Split(thumbName)
				cacheFileNames = append(cacheFileNames, thumbName)
				errorIndicationName := c.errorIndicationPath(thumbName)
				_, errorIndicationName = filepath.Split(errorIndicationName)
				cacheFileNames = append(cacheFileNames, errorIndicationName)
			}
			previewName, err := c.previewPath(fileName)
			if err == nil {
				_, previewName = filepath.Split(previewName)
				cacheFileNames = append(cacheFileNames, previewName)
				errorIndicationName := c.errorIndicationPath(previewName)
				_, errorIndicationName = filepath.Split(errorIndicationName)
				cacheFileNames = append(cacheFileNames, errorIndicationName)
			}
		}
	}

	// Compare the files in cache path with expected files
	fileInfos, _ := os.ReadDir(fullCachePath)
	nbrRemovedFiles := 0
	for _, fileInfo := range fileInfos {
		if !contains(cacheFileNames, fileInfo.Name()) {
			filePath := filepath.Join(fullCachePath, fileInfo.Name())
			log.Debug("Removing ", filePath)
			os.RemoveAll(filePath)
			nbrRemovedFiles++
		}
	}
	return nbrRemovedFiles
}
