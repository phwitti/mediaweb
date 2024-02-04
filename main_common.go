package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func mainCommon() *WebAPI {
	s := loadSettings(findConfFile())
	log.SetLevel(s.logLevel)
	if s.logFile != "" {
		log.Info("Logging will continue in file ", s.logFile)
		file, err := os.OpenFile(s.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Panic("Failed to create logfile ", s.logFile)
		}
		log.SetOutput(file)
		defer file.Close()
	}
	log.Info("Version: ", applicationVersion)
	log.Info("Build time: ", applicationBuildTime)
	log.Info("Git hash: ", applicationGitHash)
	media := createMedia(s.mediaPath, s.cachePath,
		s.enableThumbCache, s.ignoreExifThumbs, s.genThumbsOnStartup,
		s.genThumbsOnAdd, s.genAlbumThumbs, s.autoRotate, s.enablePreview, s.previewMaxSide,
		s.genPreviewForSmallImages, s.genPreviewOnStartup, s.genPreviewOnAdd,
		s.enableCacheCleanup)
	webAPI := CreateWebAPI(s.port, s.ip, "templates", media,
		s.userName, s.password, s.tlsCertFile, s.tlsKeyFile)
	return webAPI
}

// getFullPath returns the full path from an absolute base
// path and a relative path. Returns error on security hacks,
// i.e. when someone tries to access ../../../ for example to
// get files that are not within configured base path.
//
// Always returning front slashes / as path separator
func getFullPath(basePath, relativePath string) (string, error) {
	fullPath := filepath.ToSlash(filepath.Join(basePath, relativePath))
	diffPath, err := filepath.Rel(basePath, fullPath)
	diffPath = filepath.ToSlash(diffPath)
	if err != nil || strings.HasPrefix(diffPath, "../") {
		return basePath, fmt.Errorf("hacker attack, someone tries to access: %s", fullPath)
	}
	return fullPath, nil
}
