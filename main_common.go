package main

import (
	"os"

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
		s.genThumbsOnAdd, s.autoRotate, s.enablePreview, s.previewMaxSide,
		s.genPreviewOnStartup, s.genPreviewOnAdd, s.enableCacheCleanup)
	webAPI := CreateWebAPI(s.port, s.ip, "templates", media,
		s.userName, s.password, s.tlsCertFile, s.tlsKeyFile)
	return webAPI
}
