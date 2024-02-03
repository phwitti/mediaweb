package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

type settings struct {
	port                     int       // Network port
	ip                       string    // Network IP ("" means any)
	mediaPath                string    // Top level path for media files
	cachePath                string    // Top level path for cache (thumbs and preview)
	enableThumbCache         bool      // Generate thumbnails
	ignoreExifThumbs         bool      // Ignore embedded exif thumbnails
	genThumbsOnStartup       bool      // Generate all thumbnails on startup
	genThumbsOnAdd           bool      // Generate thumbnails when file added (start watcher)
	autoRotate               bool      // Rotate JPEG files when needed
	enablePreview            bool      // Generate preview files
	previewMaxSide           int       // Max height/width of preview file
	genPreviewForSmallImages bool      // Generate preview files also for images smaller then previewMaxSide
	genPreviewOnStartup      bool      // Generate all preview on startup
	genPreviewOnAdd          bool      // Generate preview when file added (start watcher)
	enableCacheCleanup       bool      // Clear cache from unnecessary files
	logLevel                 log.Level // Logging level
	logFile                  string    // Log file ("" means stderr)
	userName                 string    // User name ("" means no authentication)
	password                 string    // Password
	tlsCertFile              string    // TLS certification file
	tlsKeyFile               string    // TLS key file
}

// defaultConfPath holds configuration file paths in priority order
var defaultConfPaths = []string{"mediaweb.conf", "/etc/mediaweb.conf", "/etc/mediaweb/mediaweb.conf"}

// For unit test purposes we do it like this (to be able to change confPaths)
var confPaths = defaultConfPaths

// findConfFile finds the location of the configuration file depending on confPaths
// panics if no configuration file was found
func findConfFile() string {
	result := ""
	for _, path := range confPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			result = path
			break
		}
	}
	if result == "" {
		log.Panic("No configuration file found. Looked in ", strings.Join(confPaths, ", "))
	}
	return result
}

// loadSettings loads settings from a .conf file. Panics if configuration file
// don't exist or if any of the mandatory settings don't exist.
func loadSettings(fileName string) settings {
	result := settings{}
	log.Info("Loading configuration: ", fileName)
	config, err := ini.Load(fileName)
	if err != nil {
		log.Panic(err)
	}

	section, err := config.GetSection("")
	if err != nil {
		log.Panic(err)
	}

	// Load port (MANDATORY)
	if !section.HasKey("port") {
		log.Panic("Mandatory property 'port' is not defined in ", fileName)
	}
	port, err := section.Key("port").Int()
	if err != nil {
		log.Panic(err)
	}
	result.port = port

	// Load IP (OPTIONAL)
	// Default: ""
	ip := section.Key("ip").MustString("")
	result.ip = ip

	// Load mediaPath (MANDATORY)
	if !section.HasKey("mediapath") {
		log.Panic("Mandatory property 'mediapath' is not defined in ", fileName)
	}
	mediaPath := section.Key("mediapath").MustString("")
	result.mediaPath = mediaPath

	// Load cachePath (OPTIONAL)
	// Default: OS temp directory
	if section.HasKey("cachepath") {
		cachePath := section.Key("cachepath").MustString("")
		result.cachePath = cachePath
	} else {
		// For backwards compatibility with old versions
		if section.HasKey("thumbpath") {
			cachePath := section.Key("thumbpath").MustString("")
			result.cachePath = cachePath
		} else {
			// Use default temporary directory + mediaweb
			tempDir := os.TempDir()
			result.cachePath = filepath.Join(tempDir, "mediaweb")
		}
	}

	// Check that mediapath and cachepath are not the same
	if pathEquals(result.mediaPath, result.cachePath) {
		log.Panicf("cachepath and mediapath have the same value '%s'", result.mediaPath)
	}

	// Load enableThumbCache (OPTIONAL)
	// Default: true
	result.enableThumbCache = readOptionalBool(section, "enablethumbcache", true)

	// Load ignoreExifThumbs (OPTIONAL)
	// Default: false
	result.ignoreExifThumbs = readOptionalBool(section, "ignoreexifthumbs", false)

	// Load genthumbsonstartup (OPTIONAL)
	// Default: false
	result.genThumbsOnStartup = readOptionalBool(section, "genthumbsonstartup", false)

	// Load genthumbsonadd (OPTIONAL)
	// Default: true
	result.genThumbsOnAdd = readOptionalBool(section, "genthumbsonadd", true)

	// Load autoRotate (OPTIONAL)
	// Default: true
	result.autoRotate = readOptionalBool(section, "autorotate", true)

	// Load enablePreview (OPTIONAL)
	// Default: false
	result.enablePreview = readOptionalBool(section, "enablepreview", false)

	// Load previewMaxSide (OPTIONAL)
	// Default: 1280 (pixels)
	result.previewMaxSide = readOptionalInt(section, "previewmaxside", 1280)

	// Load genPreviewForSmallImages (OPTIONAL)
	// Default: false
	result.genPreviewForSmallImages = readOptionalBool(section, "genpreviewforsmallimages", false)

	// Load genpreviewonstartup (OPTIONAL)
	// Default: false
	result.genPreviewOnStartup = readOptionalBool(section, "genpreviewonstartup", false)

	// Load genpreviewonadd (OPTIONAL)
	// Default: true
	result.genPreviewOnAdd = readOptionalBool(section, "genpreviewonadd", true)

	// Load enableCacheCleanup (OPTIONAL)
	// Default: false
	result.enableCacheCleanup = readOptionalBool(section, "enablecachecleanup", false)

	// Load logFile (OPTIONAL)
	// Default: "" (log to stderr)
	logFile := section.Key("logfile").MustString("")
	result.logFile = logFile

	// Load logLevel (OPTIONAL)
	// Default: info
	logLevel := section.Key("loglevel").MustString("info")
	result.logLevel = toLogLvl(logLevel)

	// Load username (OPTIONAL)
	// Default: "" (no authentication)
	userName := section.Key("username").MustString("")
	result.userName = userName

	// Load password (OPTIONAL)
	// Default: ""
	password := section.Key("password").MustString("")
	result.password = password

	// Load tlsCertFile (OPTIONAL)
	// Default: ""
	tlsCertFile := section.Key("tlscertfile").MustString("")
	result.tlsCertFile = tlsCertFile

	// Load tlsKeyFile (OPTIONAL)
	// Default: ""
	tlsKeyFile := section.Key("tlskeyfile").MustString("")
	result.tlsKeyFile = tlsKeyFile

	return result
}

func toLogLvl(level string) log.Level {
	var logLevel log.Level
	switch level {
	case "trace":
		logLevel = log.TraceLevel
	case "debug":
		logLevel = log.DebugLevel
	case "info":
		logLevel = log.InfoLevel
	case "warn":
		logLevel = log.WarnLevel
	case "error":
		logLevel = log.ErrorLevel
	case "panic":
		logLevel = log.PanicLevel
	default:
		log.Warnf("Invalid loglevel '%s'. Using info level.", level)
		logLevel = log.InfoLevel
	}

	return logLevel
}

func pathEquals(path1, path2 string) bool {
	diffPath, err := filepath.Rel(path1, path2)
	if err == nil && (diffPath == "" || diffPath == ".") {
		return true
	}
	return false
}

func readOptionalBool(section *ini.Section, key string, defaultVal bool) bool {
	if !section.HasKey(key) {
		return defaultVal
	}

	result, err := section.Key(key).Bool()
	if err != nil {
		result = defaultVal
		log.Warn(err)
	}
	return result
}

func readOptionalInt(section *ini.Section, key string, defaultVal int) int {
	if !section.HasKey(key) {
		return defaultVal
	}

	result, err := section.Key(key).Int()
	if err != nil {
		result = defaultVal
		log.Warn(err)
	}
	return result
}
