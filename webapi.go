package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// WebAPI represents the REST API server.
type WebAPI struct {
	server       *http.Server
	templatePath string // Path to the templates
	media        *Media
	userName     string // User name ("" means no authentication)
	password     string // Password
	tlsCertFile  string // TLS certification file ("" means no TLS)
	tlsKeyFile   string // TLS key file ("" means no TLS)
}

// CreateWebAPI creates a new Web API instance
func CreateWebAPI(port int, ip, templatePath string, media *Media, userName, password,
	tlsCertFile, tlsKeyFile string) *WebAPI {
	portStr := fmt.Sprintf("%s:%d", ip, port)
	server := &http.Server{Addr: portStr}
	webAPI := &WebAPI{
		server:       server,
		templatePath: templatePath,
		media:        media,
		userName:     userName,
		password:     password,
		tlsCertFile:  tlsCertFile,
		tlsKeyFile:   tlsKeyFile}
	http.Handle("/", webAPI)
	return webAPI
}

// Start starts the HTTP server. Stop it using the Stop function. Non-blocking.
// Returns a channel that is written to when the HTTP server has stopped.
func (wa *WebAPI) Start() chan bool {
	done := make(chan bool)

	go func() {
		log.Info("Starting Web API on port ", wa.server.Addr)
		if wa.tlsCertFile != "" && wa.tlsKeyFile != "" {
			log.Info("Using TLS (HTTPS)")
			if err := wa.server.ListenAndServeTLS(wa.tlsCertFile, wa.tlsKeyFile); err != nil {
				// cannot panic, because this probably is an intentional close
				log.Info("WebAPI: ListenAndServeTLS() shutdown reason: ", err)
			}
		} else {
			if err := wa.server.ListenAndServe(); err != nil {
				// cannot panic, because this probably is an intentional close
				log.Info("WebAPI: ListenAndServeTLS() shutdown reason: ", err)
			}
		}
		// TODO fix this wa.media.stopWatcher() // Stop the folder watcher (if it is running)
		done <- true // Signal that http server has stopped
	}()
	return done
}

// Stop stops the HTTP server.
func (wa *WebAPI) Stop() {
	wa.server.Shutdown(context.Background())
}

// ServeHTTP handles incoming HTTP requests
func (wa *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Handle authentication
	if wa.userName != "" {
		// Authentication required
		user, pass, _ := r.BasicAuth()
		if wa.userName != user || wa.password != pass {
			log.Infof("Invalid user login attempt. user: %s, password: %s", user, pass)
			w.Header().Set("WWW-Authenticate", "Basic realm=\"MediaWEB requires username and password\"")
			http.Error(w, "Unauthorized. Invalid username or password.", http.StatusUnauthorized)
			return
		}
	}

	// Handle request
	var head string
	originalURL := r.URL.Path
	log.Trace("Got request: ", r.URL.Path)
	head, r.URL.Path = shiftPath(r.URL.Path)
	if head == "shutdown" && r.Method == "POST" {
		wa.Stop()
	} else if head == "folder" && r.Method == "GET" {
		wa.serveHTTPFolder(w, r)
	} else if head == "media" && r.Method == "GET" {
		wa.serveHTTPMedia(w, r)
	} else if head == "thumb" && r.Method == "GET" {
		wa.serveHTTPThumbnail(w, r)
	} else if head == "isPreCacheInProgress" && r.Method == "GET" {
		toJSON(w, wa.media.isPreCacheInProgress())
	} else if r.Method == "GET" {
		r.URL.Path = originalURL
		wa.serveHTTPStatic(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "This is not a valid path: %s or method %s!", r.URL.Path, r.Method)
	}
}

func (wa *WebAPI) serveHTTPStatic(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Path
	if len(r.URL.Path) > 0 {
		fileName = r.URL.Path[1:] // Remove '/'
	}
	if fileName == "" {
		// Default is index page
		fileName = "index.html"
	}

	bytes, err := embedStaticContent.ReadFile("templates/" + fileName)
	if err != nil || len(bytes) == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Unable to find: %s!", fileName)
	} else {
		if filepath.Ext(fileName) == ".html" {
			w.Header().Set("Content-Type", "text/html")
		} else if filepath.Ext(fileName) == ".ico" {
			w.Header().Set("Content-Type", "image/x-icon")
		} else {
			w.Header().Set("Content-Type", "image/png")
		}
		w.Write(bytes)
	}
}

// serveHTTPFolder generates JSON will files in folder
func (wa *WebAPI) serveHTTPFolder(w http.ResponseWriter, r *http.Request) {
	folder := ""
	if len(r.URL.Path) > 0 {
		folder = r.URL.Path[1:] // Remove '/'
	}
	files, err := wa.media.getFiles(folder)
	if err != nil {
		http.Error(w, "Get files: "+err.Error(), http.StatusNotFound)
		return
	}
	toJSON(w, files)
}

// serveHTTPMedia opens the media
func (wa *WebAPI) serveHTTPMedia(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Path
	// Only accept media files of security reasons
	if getFileType(relativePath) == "" {
		http.Error(w, "Not a valid media file: "+relativePath, http.StatusNotFound)
		return
	}
	originalImage, hasOriginalImageQuery := r.URL.Query()["original-image"]
	// Write preview file if possible and allowed
	if !hasOriginalImageQuery || originalImage[0] != "true" {
		err := wa.media.writePreview(w, relativePath)
		if err == nil {
			// Previews are always in JPEG format
			w.Header().Set("Content-Type", "image/jpeg")
			return
		}
	}
	if wa.media.isRotationNeeded(relativePath) {
		// This is a JPEG file which requires rotation.
		w.Header().Set("Content-Type", "image/jpeg")
		err := wa.media.rotateAndWrite(w, relativePath)
		if err != nil {
			http.Error(w, "Rotate file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// This is any other media file
		fullPath, err := wa.media.getFullMediaPath(relativePath)
		if err != nil {
			http.Error(w, "Get files: "+err.Error(), http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, fullPath)
	}
}

// serveHTTPThumbnail opens the media thumbnail or the default thumbnail
// if no thumbnail exist.
func (wa *WebAPI) serveHTTPThumbnail(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Path
	err := wa.media.writeThumbnail(w, relativePath)
	if err == nil {
		w.Header().Set("Content-Type", "image/jpeg")
	} else {
		// No thumbnail. Use the default
		w.Header().Set("Content-Type", "image/png")
		fileType := getFileType(relativePath)
		if fileType == "image" {
			w.Write(embedImageIconBytes)
			//http.ServeFile(w, r, wa.templatePath+"/icon_image.png")
		} else if fileType == "video" {
			w.Write(embedVideoIconBytes)
			//http.ServeFile(w, r, wa.templatePath+"/icon_video.png")
		} else {
			// Folder
			w.Write(embedFolderIconBytes)
			//http.ServeFile(w, r, wa.templatePath+"/icon_folder.png")
		}
	}
}

// toJSON converts the v object to JSON and writes result to the response
func toJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// shiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}
