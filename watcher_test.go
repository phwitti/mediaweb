package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func copyFile(t *testing.T, sourceFile, destinationFile string) {
	t.Helper()
	input, err := os.ReadFile(sourceFile)
	assertExpectNoErr(t, "", err)

	err = os.WriteFile(destinationFile, input, 0644)
	assertExpectNoErr(t, "", err)
}

// copyFileExternal will perform the copy from an external process,
// which will lock the file. This is a more realistic scenario
// than copyFile.
func copyFileExternal(t *testing.T, sourceFile, destinationFile string) {
	t.Helper()
	var cpCmd string
	var cpArgs []string
	if os.PathSeparator == '\\' {
		// Hacky way to check that this is Windows
		cpCmd = "cmd"
		cpArgs = []string{
			"/C",
			"start",
			"/B",
			"cmd",
			"/C",
			"copy",
			filepath.FromSlash(sourceFile),
			filepath.FromSlash(destinationFile)}
	} else {
		// This is linux (or mac)
		cpCmd = "cp"
		cpArgs = []string{
			sourceFile,
			destinationFile}
	}
	cmd := exec.Command(cpCmd, cpArgs...)
	err := cmd.Run()
	assertExpectNoErr(t, "", err)
}

// assertFileCreated checks if a file is created within 10 seconds or fails.
func assertFileCreated(t *testing.T, message string, name string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(name); err == nil {
			return // File found
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("File %s not created. %s", name, message)
}

// assertFileRemoved check if a file is removed within 10 seconds or fails.
func assertFileRemoved(t *testing.T, message string, name string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(name); err != nil {
			return // File removed
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("File %s was never removed. %s", name, message)
}

func TestWatcherImages(t *testing.T) {
	mediaPath := "tmpout/TestWatcherImages"
	os.RemoveAll(mediaPath)
	os.MkdirAll(mediaPath, os.ModePerm)

	cache := "tmpcache/TestWatcherImages"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia(mediaPath, cache, true, false, false, true, true, true, false, 0, false, false, false, true)
	defer media.watcher.stopWatcherAndWait()

	time.Sleep(100 * time.Millisecond) // Wait for watcher to start

	// Add a new file
	copyFile(t, "templates/icon_image.png", mediaPath+"/icon_image.png")

	// Verify that thumbnail was created
	assertFileCreated(t, "", cache+"/icon_image.thumb.jpg")

	// Remove file
	os.Remove(mediaPath + "/icon_image.png")

	// Verify that thumbnail was removed
	assertFileRemoved(t, "", cache+"/icon_image.thumb.jpg")

	// Add many files
	copyFile(t, "templates/icon_image.png", mediaPath+"/icon_image.png")
	copyFile(t, "testmedia/exif_rotate/no_exif.jpg", mediaPath+"/no_exif.jpg")
	copyFile(t, "testmedia/gif.gif", mediaPath+"/gif.gif")
	copyFile(t, "testmedia/tiff.tiff", mediaPath+"/tiff.tiff")

	// Verify that thumbnails where created
	assertFileCreated(t, "", cache+"/icon_image.thumb.jpg")
	assertFileCreated(t, "", cache+"/no_exif.thumb.jpg")
	assertFileCreated(t, "", cache+"/gif.thumb.jpg")
	assertFileCreated(t, "", cache+"/tiff.thumb.jpg")

}

func TestGetDir(t *testing.T) {
	assertEqualsStr(t, "", ".", getDir("."))
	assertEqualsStr(t, "", ".", getDir("watcher.go"))
	assertEqualsStr(t, "", "testmedia", getDir("testmedia"))
	assertEqualsStr(t, "", "testmedia", getDir("testmedia/gif.gif"))
	assertEqualsStr(t, "", "testmedia/exif_rotate", getDir("testmedia/exif_rotate"))
	assertEqualsStr(t, "", "testmedia/exif_rotate", getDir("testmedia/exif_rotate/normal.jpg"))
}

func TestWatcherFileLocked(t *testing.T) {
	mediaPath := "tmpout/TestWatcherFileLocked"
	os.RemoveAll(mediaPath)
	os.MkdirAll(mediaPath, os.ModePerm)

	cache := "tmpcache/TestWatcherFileLocked"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia(mediaPath, cache, true, false, false, true, true, true, false, 0, false, false, false, false)
	defer media.watcher.stopWatcherAndWait()

	time.Sleep(100 * time.Millisecond) // Wait for watcher to start

	// Add many files
	copyFileExternal(t, "templates/icon_image.png", mediaPath+"/icon_image.png")
	copyFileExternal(t, "testmedia/exif_rotate/no_exif.jpg", mediaPath+"/no_exif.jpg")
	copyFileExternal(t, "testmedia/gif.gif", mediaPath+"/gif.gif")
	copyFileExternal(t, "testmedia/tiff.tiff", mediaPath+"/tiff.tiff")

	// Verify that thumbnails where created
	assertFileCreated(t, "", cache+"/icon_image.thumb.jpg")
	assertFileCreated(t, "", cache+"/no_exif.thumb.jpg")
	assertFileCreated(t, "", cache+"/gif.thumb.jpg")
	assertFileCreated(t, "", cache+"/tiff.thumb.jpg")

}

func TestWatcherSubfolder(t *testing.T) {
	mediaPath := "tmpout/TestWatcherSubfolder"
	os.RemoveAll(mediaPath)
	os.MkdirAll(mediaPath, os.ModePerm)

	cache := "tmpcache/TestWatcherSubfolder"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia(mediaPath, cache, true, false, false, true, true, true, false, 0, false, false, false, true)
	defer media.watcher.stopWatcherAndWait()

	time.Sleep(100 * time.Millisecond) // Wait for watcher to start

	// Add a subdirectory with files
	os.MkdirAll(mediaPath+"/subdir", os.ModePerm)
	time.Sleep(500 * time.Millisecond) // Wait for subfolder to be watched
	copyFile(t, "templates/icon_image.png", mediaPath+"/subdir/icon_image.png")
	copyFile(t, "testmedia/exif_rotate/no_exif.jpg", mediaPath+"/subdir/no_exif.jpg")
	copyFile(t, "testmedia/gif.gif", mediaPath+"/subdir/gif.gif")
	copyFile(t, "testmedia/tiff.tiff", mediaPath+"/subdir/tiff.tiff")

	// Verify that thumbnails where created for subdirectory
	assertFileCreated(t, "", cache+"/subdir/icon_image.thumb.jpg")
	assertFileCreated(t, "", cache+"/subdir/no_exif.thumb.jpg")
	assertFileCreated(t, "", cache+"/subdir/gif.thumb.jpg")
	assertFileCreated(t, "", cache+"/subdir/tiff.thumb.jpg")

	// Add a subdirectory of the subdiretory
	os.MkdirAll(mediaPath+"/subdir/submore", os.ModePerm)
	time.Sleep(500 * time.Millisecond) // Wait for subfolder to be watched
	copyFile(t, "testmedia/exif_rotate/no_exif.jpg", mediaPath+"/subdir/submore/no_exif.jpg")
	assertFileCreated(t, "", cache+"/subdir/submore/no_exif.thumb.jpg")

	// Remove directory
	os.RemoveAll(mediaPath + "/subdir/submore")

	// Verify that directory was removed
	assertFileRemoved(t, "", cache+"/subdir/submore")

	// But secure that other files are kept
	assertFileCreated(t, "", cache+"/subdir/icon_image.thumb.jpg")
}

func TestWatcherVideo(t *testing.T) {
	mediaPath := "tmpout/TestWatcherVideo"
	os.RemoveAll(mediaPath)
	os.MkdirAll(mediaPath, os.ModePerm)

	cache := "tmpcache/TestWatcherVideo"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia(mediaPath, cache, true, false, false, true, true, true, false, 0, false, false, false, false)
	defer media.watcher.stopWatcherAndWait()

	if !hasVideoThumbnailSupport() {
		t.Skip("ffmpeg not installed skipping test")
		return
	}

	time.Sleep(100 * time.Millisecond) // Wait for watcher to start

	// Add a new video file
	copyFile(t, "testmedia/video.mp4", mediaPath+"/video.mp4")

	// Verify that thumbnail was created
	assertFileCreated(t, "", cache+"/video.thumb.jpg")
}

func TestWatchFolder(t *testing.T) {
	// Don't start the watcher, so that we can test its internal
	// functionality
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)

	watcher, err := fsnotify.NewWatcher()
	assertExpectNoErr(t, "", err)

	// Test some valid
	err = media.watcher.watchFolder(watcher, "testmedia")
	assertExpectNoErr(t, "", err)
	err = media.watcher.watchFolder(watcher, "templates")
	assertExpectNoErr(t, "", err)

	// Test some invalid
	err = media.watcher.watchFolder(watcher, "dontexist")
	assertExpectErr(t, "", err)
	err = media.watcher.watchFolder(watcher, "testmedia/dontexist")
	assertExpectErr(t, "", err)
	err = media.watcher.watchFolder(watcher, "testmedia/jpeg.jpg")
	assertExpectErr(t, "", err)
}
