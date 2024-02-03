package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cozy/goexif2/exif"
	"github.com/disintegration/imaging"
	log "github.com/sirupsen/logrus"
)

var imgExtensions = [...]string{".png", ".jpg", ".jpeg", ".tif", ".tiff", ".gif"}
var vidExtensions = [...]string{".avi", ".mov", ".vid", ".mkv", ".mp4"}

// Media represents the media including its base path
type Media struct {
	mediaPath          string // Top level path for media files
	cachepath          string // Top level path for thumbnails
	enableThumbCache   bool   // Generate thumbnails
	ignoreExifThumbs   bool   // Ignore embedded exif thumbnails
	autoRotate         bool   // Rotate JPEG files when needed
	enablePreview      bool   // Resize images before provide to client
	previewMaxSide     int    // Maximum width or hight of preview image
	enableCacheCleanup bool   // Enable cleanup of cache area
	preCacheInProgress bool   // True if thumbnail/preview generation in progress
	cache              *Cache
	watcher            *Watcher // The media watcher
}

// File represents a folder or any other file
type File struct {
	Type string // folder, image or video
	Name string
	Path string // Including Name. Always using / (even on Windows)
}

// createMedia creates a new media. If thumb cache is enabled the path is
// created when needed.
func createMedia(mediaPath string, cachepath string, enableThumbCache bool, ignoreExifThumbs bool,
	genThumbsOnStartup bool, genThumbsOnAdd bool, autoRotate bool, enablePreview bool,
	previewMaxSide int, genPreviewOnStartup bool, genPreviewOnAdd bool, enabledCacheCleanup bool) *Media {
	log.Info("Media path: ", mediaPath)
	if enableThumbCache || enablePreview {
		directory := filepath.Dir(cachepath)
		err := os.MkdirAll(directory, os.ModePerm)
		if err != nil {
			log.Warnf("Unable to create cache path %s. Reason: %s", cachepath, err)
			log.Info("Thumbnail and preview cache will be disabled")
			enableThumbCache = false
			enablePreview = false
		} else {
			log.Info("Cache path: ", cachepath)
		}
	} else {
		log.Info("Cache disabled")
	}
	log.Info("JPEG auto rotate: ", autoRotate)
	log.Infof("Image preview: %t  (max width/height %d px)", enablePreview, previewMaxSide)
	media := &Media{mediaPath: filepath.ToSlash(filepath.Clean(mediaPath)),
		cachepath:          filepath.ToSlash(filepath.Clean(cachepath)),
		enableThumbCache:   enableThumbCache,
		ignoreExifThumbs:   ignoreExifThumbs,
		autoRotate:         autoRotate,
		enablePreview:      enablePreview,
		previewMaxSide:     previewMaxSide,
		enableCacheCleanup: enabledCacheCleanup,
		preCacheInProgress: false}
	log.Info("Video thumbnails supported (ffmpeg installed): ", hasVideoThumbnailSupport())
	if enableThumbCache || enablePreview {
		media.cache = createCache(media)
	}
	if enableThumbCache && genThumbsOnStartup || enablePreview && genPreviewOnStartup {
		go media.generateAllCache(enableThumbCache && genThumbsOnStartup, enablePreview && genPreviewOnStartup)
	}
	if enableThumbCache && genThumbsOnAdd || enablePreview && genPreviewOnAdd {
		media.watcher = createWatcher(media, enableThumbCache && genThumbsOnAdd, enablePreview && genPreviewOnAdd)
		go media.watcher.startWatcher()
	}
	return media
}

// getFullMediaPath returns the full path of the provided path, i.e:
// media path + relative path.
func (m *Media) getFullMediaPath(relativePath string) (string, error) {
	return getFullPath(m.mediaPath, relativePath)
}

// getRelativePath returns the relative path from an absolute base
// path and a full path path. Returns error if the base path is
// not in the full path.
//
// Always returning front slashes / as path separator
func (m *Media) getRelativePath(basePath, fullPath string) (string, error) {
	relativePath, err := filepath.Rel(basePath, fullPath)
	if err == nil {
		relativePathSlash := filepath.ToSlash(relativePath)
		if strings.HasPrefix(relativePathSlash, "../") {
			return "", fmt.Errorf("%s is not a sub-path of %s", fullPath, basePath)
		}
		return relativePathSlash, nil
	}
	return "", err
}

// getRelativeMediaPath returns the relative media path of the provided path, i.e:
// full path - media path.
func (m *Media) getRelativeMediaPath(fullPath string) (string, error) {
	return m.getRelativePath(m.mediaPath, fullPath)
}

// getFiles returns a slice of File's sorted on file name
func (m *Media) getFiles(relativePath string) ([]File, error) {
	//var files []File
	files := make([]File, 0, 500)
	fullPath, err := m.getFullMediaPath(relativePath)
	if err != nil {
		return files, err
	}
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

func (m *Media) isJPEG(pathAndFile string) bool {
	extension := filepath.Ext(pathAndFile)
	return strings.EqualFold(extension, ".jpg") ||
		strings.EqualFold(extension, ".jpeg")
}

func (m *Media) extractEXIF(relativeFilePath string) *exif.Exif {
	fullFilePath, err := m.getFullMediaPath(relativeFilePath)
	if err != nil {
		log.Info("Unable to get full media path for ", relativeFilePath)
		return nil
	}
	if !m.isJPEG(fullFilePath) {
		return nil // Only JPEG has EXIF
	}
	efile, err := os.Open(fullFilePath)
	if err != nil {
		log.Warnf("Could not open file for EXIF decoding. File: %s reason: %s", fullFilePath, err)
		return nil
	}
	defer efile.Close()
	ex, err := exif.Decode(efile)
	if err != nil {
		log.Debugf("No EXIF. file %s reason: %s", fullFilePath, err)
		return nil
	}
	return ex
}

// isRotationNeeded returns true if the file needs to be rotated.
// It finds this out by reading the EXIF rotation information
// in the file.
// If Media.autoRotate is false this function will always return
// false.
func (m *Media) isRotationNeeded(relativeFilePath string) bool {
	if !m.autoRotate {
		return false
	}
	ex := m.extractEXIF(relativeFilePath)
	if ex == nil {
		return false // No EXIF info exist
	}
	orientTag, _ := ex.Get(exif.Orientation)
	if orientTag == nil {
		return false // No Orientation
	}
	orientInt, _ := orientTag.Int(0)
	if orientInt > 1 && orientInt < 9 {
		return true // Rotation is needed
	}
	return false
}

// rotateAndWrite opens and rotates a JPG/JPEG file according to
// EXIF rotation information. Then it writes the rotated image
// to the io.Writer. NOTE! This process requires Decoding and
// encoding of the image which takes a LOT of time (2-3 sec).
// Check if image needs rotation with isRotationNeeded first.
func (m *Media) rotateAndWrite(w io.Writer, relativeFilePath string) error {
	fullPath, err := m.getFullMediaPath(relativeFilePath)
	if err != nil {
		return err
	}

	img, err := imaging.Open(fullPath, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}
	err = imaging.Encode(w, img, imaging.JPEG)
	if err != nil {
		return err
	}
	return nil
}

// writeEXIFThumbnail extracts the EXIF thumbnail from a JPEG file
// and rotates it when needed (based on the EXIF orientation tag).
// Returns err if no thumbnail exist.
func (m *Media) writeEXIFThumbnail(w io.Writer, relativeFilePath string) error {
	ex := m.extractEXIF(relativeFilePath)
	if ex == nil {
		return fmt.Errorf("no exif info for %s", relativeFilePath)
	}
	thumbBytes, err := ex.JpegThumbnail()
	if err != nil {
		return fmt.Errorf("no exif thumbnail for %s", relativeFilePath)
	}
	orientTag, _ := ex.Get(exif.Orientation)
	if orientTag == nil {
		// No Orientation assume no rotation needed
		w.Write(thumbBytes)
		return nil
	}
	orientInt, _ := orientTag.Int(0)
	if orientInt > 1 && orientInt < 9 {
		// Rotation is needed
		img, err := imaging.Decode(bytes.NewReader(thumbBytes))
		if err != nil {
			log.Warn("Unable to decode EXIF thumbnail for ", relativeFilePath)
			w.Write(thumbBytes)
			return nil
		}
		var outImg *image.NRGBA
		switch orientInt {
		case 2:
			outImg = imaging.FlipV(img)
		case 3:
			outImg = imaging.Rotate180(img)
		case 4:
			outImg = imaging.Rotate180(imaging.FlipV(img))
		case 5:
			outImg = imaging.Rotate270(imaging.FlipV(img))
		case 6:
			outImg = imaging.Rotate270(img)
		case 7:
			outImg = imaging.Rotate90(imaging.FlipV(img))
		case 8:
			outImg = imaging.Rotate90(img)
		}
		imaging.Encode(w, outImg, imaging.JPEG)
	} else {
		// No rotation is needed
		w.Write(thumbBytes)
	}
	return nil
}

// writeThumbnail writes thumbnail for media to w.
//
// It has following sequence/priority:
//  1. Write embedded EXIF thumbnail if it exist (only JPEG)
//  2. Write a cached thumbnail file exist in cachepath
//  3. Generate a thumbnail to cache and write
//  4. If all above fails return error
func (m *Media) writeThumbnail(w io.Writer, relativeFilePath string) error {
	if !isImage(relativeFilePath) && !isVideo(relativeFilePath) {
		return fmt.Errorf("not a supported media type")
	}
	if !m.ignoreExifThumbs && m.writeEXIFThumbnail(w, relativeFilePath) == nil {
		return nil
	}
	if !m.enableThumbCache {
		return fmt.Errorf("thumbnail cache disabled")
	}

	// No EXIF, check thumb cache (and generate if necessary)
	thumbFileName, err := m.cache.generateThumbnail(m, relativeFilePath)
	if err != nil {
		return err // Logging handled in generateThumbnail
	}

	thumbFile, err := os.Open(thumbFileName)
	if err != nil {
		return err
	}
	defer thumbFile.Close()

	_, err = io.Copy(w, thumbFile)
	if err != nil {
		return err
	}

	return nil
}

// getImageWidthAndHeight returns the width and height of an image.
// Returns error if the width and height could not be determined.
func (m *Media) getImageWidthAndHeight(fullMediaPath string) (int, int, error) {
	img, err := imaging.Open(fullMediaPath)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open image %s, reason: %s", fullMediaPath, err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy(), nil
}

// writePreview writes preview image for media to w.
//
// It has following sequence/priority:
//  1. Write a cached preview file exist
//  2. Generate a preview in cache and write
//  3. If all above fails return error
func (m *Media) writePreview(w io.Writer, relativeFilePath string) error {
	if !isImage(relativeFilePath) {
		return fmt.Errorf("only images support preview")
	}
	if !m.enablePreview {
		return fmt.Errorf("preview disabled")
	}

	// Check preview cache (and generate if necessary)
	previewFileName, _, err := m.cache.generatePreview(m, relativeFilePath)
	if err != nil {
		return err // Logging handled in generatePreview
	}

	previewFile, err := os.Open(previewFileName)
	if err != nil {
		return err
	}
	defer previewFile.Close()

	_, err = io.Copy(w, previewFile)
	if err != nil {
		return err
	}

	return nil
}

// PreCacheStatistics statistics results from generateCache
type PreCacheStatistics struct {
	NbrOfFolders            int
	NbrOfImages             int
	NbrOfVideos             int
	NbrOfExif               int
	NbrOfImageThumb         int
	NbrOfVideoThumb         int
	NbrOfImagePreview       int
	NbrOfFailedFolders      int // I.e. unable to list contents of folder
	NbrOfFailedImageThumb   int
	NbrOfFailedVideoThumb   int
	NbrOfFailedImagePreview int
	NbrOfSmallImages        int // Don't require any preview
	NbrRemovedCacheFiles    int
}

func (m *Media) isPreCacheInProgress() bool {
	return m.preCacheInProgress
}

func (m *Media) generateCache(relativePath string, recursive bool, thumbnails bool, preview bool) *PreCacheStatistics {
	return m.updateCache(m.cache, relativePath, recursive, thumbnails, preview)
}

// updateCache recursively (optional) goes through all files
// relativePath and its subdirectories and generates thumbnails and
// previews for these. If relativePath is "" it means generate for all files.
func (m *Media) updateCache(c *Cache, relativePath string, recursive bool, thumbnails bool, preview bool) *PreCacheStatistics {
	prevProgress := m.preCacheInProgress
	m.preCacheInProgress = true
	defer func() { m.preCacheInProgress = prevProgress }()

	stat := PreCacheStatistics{}
	files, err := m.getFiles(relativePath)
	if err != nil {
		stat.NbrOfFailedFolders = 1
		return &stat
	}
	for _, file := range files {
		if file.Type == "folder" {
			if recursive {
				stat.NbrOfFolders++
				newStat := m.updateCache(c, file.Path, true, thumbnails, preview) // Recursive
				stat.NbrOfFolders += newStat.NbrOfFolders
				stat.NbrOfImages += newStat.NbrOfImages
				stat.NbrOfVideos += newStat.NbrOfVideos
				stat.NbrOfExif += newStat.NbrOfExif
				stat.NbrOfImageThumb += newStat.NbrOfImageThumb
				stat.NbrOfVideoThumb += newStat.NbrOfVideoThumb
				stat.NbrOfImagePreview += newStat.NbrOfImagePreview
				stat.NbrOfFailedFolders += newStat.NbrOfFailedFolders
				stat.NbrOfFailedImageThumb += newStat.NbrOfFailedImageThumb
				stat.NbrOfFailedVideoThumb += newStat.NbrOfFailedVideoThumb
				stat.NbrOfFailedImagePreview += newStat.NbrOfFailedImagePreview
				stat.NbrOfSmallImages += newStat.NbrOfSmallImages
				stat.NbrRemovedCacheFiles += newStat.NbrRemovedCacheFiles
			}
		} else {
			if file.Type == "image" {
				stat.NbrOfImages++
			} else if file.Type == "video" {
				stat.NbrOfVideos++
			}
			// Check if file has EXIF thumbnail
			hasExifThumb := false
			if !m.ignoreExifThumbs {
				ex := m.extractEXIF(file.Path)
				if ex != nil {
					_, err := ex.JpegThumbnail()
					if err == nil {
						// Media has EXIF thumbnail
						stat.NbrOfExif++
						hasExifThumb = true
					}
				}
			}
			if thumbnails && !hasExifThumb && !c.hasThumbnail(file.Path) {
				// Generate new thumbnail
				_, err = c.generateThumbnail(m, file.Path)
				if err != nil {
					if file.Type == "image" {
						stat.NbrOfFailedImageThumb++
					} else if file.Type == "video" {
						stat.NbrOfFailedVideoThumb++
					}
				} else {
					if file.Type == "image" {
						stat.NbrOfImageThumb++
					} else if file.Type == "video" {
						stat.NbrOfVideoThumb++
					}
				}
			}
			if preview && file.Type == "image" && !c.hasPreview(file.Path) {
				// Generate new preview
				_, tooSmall, err := c.generatePreview(m, file.Path)
				if err != nil {
					if tooSmall {
						stat.NbrOfSmallImages++
					} else {
						stat.NbrOfFailedImagePreview++
					}
				} else {
					stat.NbrOfImagePreview++
				}
			}
		}
	}
	if m.enableCacheCleanup {
		stat.NbrRemovedCacheFiles += c.cleanupCache(relativePath, files)
	}
	return &stat
}

// generateAllCache goes through all files in the media path
// and generates thumbnails/preview for these
func (m *Media) generateAllCache(thumbnails, preview bool) {
	log.Infof("Pre-generating cache (thumbnails: %t, preview: %t)", thumbnails, preview)
	startTime := time.Now().UnixNano()
	stat := m.generateCache("", true, thumbnails, preview)
	deltaTime := (time.Now().UnixNano() - startTime) / int64(time.Second)
	minutes := int(deltaTime / 60)
	seconds := int(deltaTime) - minutes*60
	log.Infof("Generating cache took %d minutes and %d seconds", minutes, seconds)
	log.Info("Number of folders: ", stat.NbrOfFolders)
	log.Info("Number of images: ", stat.NbrOfImages)
	log.Info("Number of videos: ", stat.NbrOfVideos)
	log.Info("Number of images with embedded EXIF: ", stat.NbrOfExif)
	log.Info("Number of generated image thumbnails: ", stat.NbrOfImageThumb)
	log.Info("Number of generated video thumbnails: ", stat.NbrOfVideoThumb)
	log.Info("Number of generated image previews: ", stat.NbrOfImagePreview)
	log.Info("Number of failed folders: ", stat.NbrOfFailedFolders)
	log.Info("Number of failed image thumbnails: ", stat.NbrOfFailedImageThumb)
	log.Info("Number of failed video thumbnails: ", stat.NbrOfFailedVideoThumb)
	log.Info("Number of failed image previews: ", stat.NbrOfFailedImagePreview)
	log.Info("Number of small images not require preview: ", stat.NbrOfSmallImages)
	log.Info("Number of removed cache files: ", stat.NbrRemovedCacheFiles)
}
