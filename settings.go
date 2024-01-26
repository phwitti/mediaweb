package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

type settings struct {
	port                int       // Network port
	ip                  string    // Network IP ("" means any)
	mediaPath           string    // Top level path for media files
	cachePath           string    // Top level path for cache (thumbs and preview)
	enableThumbCache    bool      // Generate thumbnails
	genThumbsOnStartup  bool      // Generate all thumbnails on startup
	genThumbsOnAdd      bool      // Generate thumbnails when file added (start watcher)
	autoRotate          bool      // Rotate JPEG files when needed
	enablePreview       bool      // Generate preview files
	previewMaxSide      int       // Max height/width of preview file
	genPreviewOnStartup bool      // Generate all preview on startup
	genPreviewOnAdd     bool      // Generate preview when file added (start watcher)
	enableCacheCleanup  bool      // Clear cache from unnecessary files
	logLevel            log.Level // Logging level
	logFile             string    // Log file ("" means stderr)
	userName            string    // User name ("" means no authentication)
	password            string    // Password
	tlsCertFile         string    // TLS certification file
	tlsKeyFile          string    // TLS key file
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
		log.Panic("No configuration file found. Looked in", strings.Join(confPaths, ", "))
	}
	return result
}

// loadSettings loads settings from a .conf file. Panics if configuration file
// don't exist or if any of the mandatory settings don't exist.
func loadSettings(fileName string) settings {
	result := settings{}
	log.Info("Loading configuration:", fileName)
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
		log.Panic("Mandatory property 'port' is not defined in", fileName)
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
		log.Panic("Mandatory property 'mediapath' is not defined in", fileName)
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
	enableThumbCache, err := section.Key("enablethumbcache").Bool()
	if err != nil {
		enableThumbCache = true
		log.Warn(err)
	}
	result.enableThumbCache = enableThumbCache

	// Load genthumbsonstartup (OPTIONAL)
	// Default: false
	genThumbsOnStartup, err := section.Key("genthumbsonstartup").Bool()
	if err != nil {
		genThumbsOnStartup = false
		log.Warn(err)
	}
	result.genThumbsOnStartup = genThumbsOnStartup

	// Load genthumbsonadd (OPTIONAL)
	// Default: true
	genThumbsOnAdd, err := section.Key("genthumbsonadd").Bool()
	if err != nil {
		genThumbsOnAdd = true
		log.Warn(err)
	}
	result.genThumbsOnAdd = genThumbsOnAdd

	// Load autoRotate (OPTIONAL)
	// Default: true
	autoRotate, err := section.Key("autorotate").Bool()
	if err != nil {
		autoRotate = true
		log.Warn(err)
	}
	result.autoRotate = autoRotate

	// Load enablePreview (OPTIONAL)
	// Default: false
	enablePreview, err := section.Key("enablepreview").Bool()
	if err != nil {
		enablePreview = false
		log.Warn(err)
	}
	result.enablePreview = enablePreview

	// Load previewMaxSide (OPTIONAL)
	// Default: 1280 (pixels)
	previewMaxSide, err := section.Key("previewmaxside").Int()
	if err != nil {
		previewMaxSide = 1280
		log.Warn(err)
	}
	result.previewMaxSide = previewMaxSide

	// Load genpreviewonstartup (OPTIONAL)
	// Default: false
	genPreviewOnStartup, err := section.Key("genpreviewonstartup").Bool()
	if err != nil {
		genPreviewOnStartup = false
		log.Warn(err)
	}
	result.genPreviewOnStartup = genPreviewOnStartup

	// Load genpreviewonadd (OPTIONAL)
	// Default: true
	genPreviewOnAdd, err := section.Key("genpreviewonadd").Bool()
	if err != nil {
		genPreviewOnAdd = true
		log.Warn(err)
	}
	result.genPreviewOnAdd = genPreviewOnAdd

	// Load enableCacheCleanup (OPTIONAL)
	// Default: false
	enableCacheCleanup, err := section.Key("enablecachecleanup").Bool()
	if err != nil {
		enableCacheCleanup = false
		log.Warn(err)
	}
	result.enableCacheCleanup = enableCacheCleanup

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
