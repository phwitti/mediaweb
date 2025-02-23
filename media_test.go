package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type timerType struct {
	start int64
	stop  int64
}

// For benchmark timing tests
var timer timerType

func RestartTimer() {
	timer.start = time.Now().UnixNano()
}

func LogTime(t *testing.T, whatWasMeasured string) {
	timer.stop = time.Now().UnixNano()
	deltaMs := (timer.stop - timer.start) / int64(time.Millisecond)
	t.Logf("%s %d ms", whatWasMeasured, deltaMs)
}

func TestGetFiles(t *testing.T) {
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	files, err := media.getFiles("")
	assertExpectNoErr(t, "", err)
	assertTrue(t, "No files found", len(files) > 5)
}

func TestGetFilesInvalid(t *testing.T) {
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	files, err := media.getFiles("invalidfolder")
	assertExpectErr(t, "invalid path shall give errors", err)
	assertTrue(t, "Should not find any files", len(files) == 0)
}

func TestGetFilesHacker(t *testing.T) {
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	files, err := media.getFiles("../..")
	assertExpectErr(t, "hacker path shall give errors", err)
	assertTrue(t, "Should not find any files", len(files) == 0)
}

func TestIsRotationNeeded(t *testing.T) {
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)

	rotationNeeded := media.isRotationNeeded("exif_rotate/180deg.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/mirror.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/mirror_rotate_90deg_cw.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/mirror_rotate_270deg.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/mirror_vertical.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/rotate_270deg_cw.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/rotate_90deg_cw.jpg")
	assertTrue(t, "Rotation should be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/normal.jpg")
	assertFalse(t, "Rotation should not be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("exif_rotate/no_exif.jpg")
	assertFalse(t, "Rotation should not be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("non_existing.jpg")
	assertFalse(t, "Rotation should not be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("png.png")
	assertFalse(t, "Rotation should not be needed", rotationNeeded)

	rotationNeeded = media.isRotationNeeded("../../../hackerpath/secret.jpg")
	assertFalse(t, "Rotation should not be needed", rotationNeeded)

	// Turn of rotation
	media.autoRotate = false

	rotationNeeded = media.isRotationNeeded("exif_rotate/mirror_rotate_90deg_cw.jpg")
	assertFalse(t, "Rotation should not be needed when turned off", rotationNeeded)
}

func TestRotateAndWrite(t *testing.T) {
	outFileName := "tmpout/TestRotateAndWrite/jpeg_rotated_fixed.jpg"
	os.MkdirAll("tmpout/TestRotateAndWrite", os.ModePerm) // If already exist no problem
	os.Remove(outFileName)
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	outFile, err := os.Create(outFileName)
	assertExpectNoErr(t, "unable to create out", err)
	defer outFile.Close()
	RestartTimer()
	err = media.rotateAndWrite(outFile, "jpeg_rotated.jpg")
	LogTime(t, "rotate JPG")
	assertExpectNoErr(t, "unable to rotate out", err)
	t.Logf("Manually check that %s has been rotated correctly", outFileName)
}

func tEXIFThumbnail(t *testing.T, media *Media, filename string) {
	t.Helper()
	inFileName := "exif_rotate/" + filename
	outFileName := "tmpout/TestWriteEXIFThumbnail/thumb_" + filename
	os.Remove(outFileName)
	outFile, err := os.Create(outFileName)
	assertExpectNoErr(t, "unable to create out", err)
	defer outFile.Close()
	RestartTimer()
	err = media.writeEXIFThumbnail(outFile, inFileName)
	LogTime(t, inFileName+" thumbnail time")
	assertExpectNoErr(t, "unable to extract thumbnail", err)
	assertFileExist(t, "", outFileName)
	t.Logf("Manually check that %s thumbnail is ok", outFileName)
}

func TestWriteEXIFThumbnail(t *testing.T) {
	os.MkdirAll("tmpout/TestWriteEXIFThumbnail", os.ModePerm) // If already exist no problem
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)

	tEXIFThumbnail(t, media, "normal.jpg")
	tEXIFThumbnail(t, media, "180deg.jpg")
	tEXIFThumbnail(t, media, "mirror.jpg")
	tEXIFThumbnail(t, media, "mirror_rotate_90deg_cw.jpg")
	tEXIFThumbnail(t, media, "mirror_rotate_270deg.jpg")
	tEXIFThumbnail(t, media, "mirror_vertical.jpg")
	tEXIFThumbnail(t, media, "rotate_270deg_cw.jpg")
	tEXIFThumbnail(t, media, "rotate_90deg_cw.jpg")

	// Test some invalid
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	err := media.writeEXIFThumbnail(writer, "../../../hackerpath/secret.jpg")
	assertExpectErr(t, "hacker attack shall not be allowed", err)

	err = media.writeEXIFThumbnail(writer, "no_exif.jpg")
	assertExpectErr(t, "No EXIF shall not have thumbnail", err)
}

func TestFullPath(t *testing.T) {
	// Root path
	media := createMedia(".", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	p, err := media.getFullMediaPath("afile.jpg")
	assertExpectNoErr(t, "unable to get valid full path", err)
	assertEqualsStr(t, "invalid path", "afile.jpg", p)

	_, err = media.getFullMediaPath("../../secret_file")
	assertExpectErr(t, "hackers shall not be allowed", err)

	// Relative path
	media = createMedia("arelative/path", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	p, err = media.getFullMediaPath("afile.jpg")
	assertExpectNoErr(t, "unable to get valid full path", err)
	assertEqualsStr(t, "invalid path", "arelative/path/afile.jpg", p)

	_, err = media.getFullMediaPath("../../secret_file")
	assertExpectErr(t, "hackers shall not be allowed", err)

	// Absolute path
	media = createMedia("/root/absolute/path", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	p, err = media.getFullMediaPath("afile.jpg")
	assertExpectNoErr(t, "unable to get valid full path", err)
	assertEqualsStr(t, "invalid path", "/root/absolute/path/afile.jpg", p)

	_, err = media.getFullMediaPath("../../secret_file")
	assertExpectErr(t, "hackers shall not be allowed", err)
}

func TestRelativePath(t *testing.T) {
	// Root path
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)

	result, err := media.getRelativePath("", "")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", ".", result)

	result, err = media.getRelativePath("", "directory")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "directory", result)

	// Unix slashes
	result, err = media.getRelativePath("", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir1/dir2/dir3", result)

	result, err = media.getRelativePath("dir1", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir2/dir3", result)

	result, err = media.getRelativePath("dir1/", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir2/dir3", result)

	result, err = media.getRelativePath("dir1/dir2", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir3", result)

	result, err = media.getRelativePath("dir1/dir2/", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir3", result)

	result, err = media.getRelativePath("dir1/dir2/dir3", "dir1/dir2/dir3")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", ".", result)

	// Windows slashes - this will only work on windows
	if os.PathSeparator == '\\' {
		result, err = media.getRelativePath("", "dir1\\dir2\\dir3")
		assertExpectNoErr(t, "", err)
		assertEqualsStr(t, "", "dir1/dir2/dir3", result)

		result, err = media.getRelativePath("dir1\\", "dir1\\dir2\\dir3")
		assertExpectNoErr(t, "", err)
		assertEqualsStr(t, "", "dir2/dir3", result)
	}

	// Errors
	_, err = media.getRelativePath("another", "directory")
	assertExpectErr(t, "", err)

	_, err = media.getRelativePath("/a", "b")
	assertExpectErr(t, "", err)

	// getRelativeMediaPath
	result, err = media.getRelativeMediaPath("testmedia/dir1/dir2")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "dir1/dir2", result)

	_, err = media.getRelativeMediaPath("another/dir1/dir2")
	assertExpectErr(t, "", err)
}

func TestThumbnailPath(t *testing.T) {
	media := createMedia("/c/mediapath", "/d/thumbpath", true, false, false, false, true, true, false, 0, false, false, false, false)

	thumbPath, err := media.cache.thumbnailPath("myimage.jpg")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/myimage.thumb.jpg", thumbPath)

	thumbPath, err = media.cache.thumbnailPath("subdrive/myimage.jpg")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/subdrive/myimage.thumb.jpg", thumbPath)

	thumbPath, err = media.cache.thumbnailPath("subdrive/myimage.png")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/subdrive/myimage.thumb.jpg", thumbPath)

	_, err = media.cache.thumbnailPath("subdrive/myimage")
	assertExpectErr(t, "", err)

	_, err = media.cache.thumbnailPath("subdrive/../../hacker")
	assertExpectErr(t, "", err)
}

func tGenerateImageThumbnail(t *testing.T, media *Media, inFileName, outFileName string) {
	t.Helper()
	os.Remove(outFileName)
	RestartTimer()
	err := media.cache.generateImageThumbnail(inFileName, outFileName)
	LogTime(t, inFileName+" thumbnail generation: ")
	assertExpectNoErr(t, "", err)
	assertFileExist(t, "", outFileName)
	t.Logf("Manually check that %s thumbnail is ok", outFileName)
}

func TestGenerateImageThumbnail(t *testing.T) {
	os.MkdirAll("tmpout/TestGenerateImageThumbnail", os.ModePerm) // If already exist no problem

	media := createMedia("", "", true, false, false, false, true, true, false, 0, false, false, false, false)

	tGenerateImageThumbnail(t, media, "testmedia/jpeg.jpg", "tmpout/TestGenerateImageThumbnail/jpeg_thumbnail.jpg")
	tGenerateImageThumbnail(t, media, "testmedia/jpeg_rotated.jpg", "tmpout/TestGenerateImageThumbnail/jpeg_rotated_thumbnail.jpg")
	tGenerateImageThumbnail(t, media, "testmedia/png.png", "tmpout/TestGenerateImageThumbnail/png_thumbnail.jpg")
	tGenerateImageThumbnail(t, media, "testmedia/gif.gif", "tmpout/TestGenerateImageThumbnail/gif_thumbnail.jpg")
	tGenerateImageThumbnail(t, media, "testmedia/tiff.tiff", "tmpout/TestGenerateImageThumbnail/tiff_thumbnail.jpg")
	tGenerateImageThumbnail(t, media, "testmedia/exif_rotate/no_exif.jpg", "tmpout/TestGenerateImageThumbnail/exif_rotate/no_exif.jpg")

	// Test some invalid
	err := media.cache.generateImageThumbnail("nonexisting.png", "dont_matter.png")
	assertExpectErr(t, "", err)

	err = media.cache.generateImageThumbnail("testmedia/invalid.jpg", "dont_matter.jpg")
	assertExpectErr(t, "", err)
}

func tWriteThumbnail(t *testing.T, media *Media, inFileName, outFileName string, failExpected bool) {
	t.Helper()
	os.Remove(outFileName)
	outFile, err := os.Create(outFileName)
	assertExpectNoErr(t, "unable to create out", err)
	defer outFile.Close()
	err = media.writeThumbnail(outFile, inFileName)
	if failExpected {
		assertExpectErr(t, "should fail", err)
	} else {
		assertExpectNoErr(t, "unable to write thumbnail", err)
		t.Logf("Manually check that %s thumbnail is ok", outFileName)
	}
}

func TestWriteThumbnail(t *testing.T) {
	os.RemoveAll("tmpcache/TestWriteThumbnail")
	os.MkdirAll("tmpcache/TestWriteThumbnail", os.ModePerm)
	os.RemoveAll("tmpout/TestWriteThumbnail")
	os.MkdirAll("tmpout/TestWriteThumbnail", os.ModePerm)

	media := createMedia("testmedia", "tmpcache/TestWriteThumbnail", true, false, false, false, true, true, false, 0, false, false, false, false)

	// JPEG with embedded EXIF
	tWriteThumbnail(t, media, "jpeg.jpg", "tmpout/TestWriteThumbnail/jpeg.jpg", false)

	// JPEG without embedded EXIF
	tWriteThumbnail(t, media, "exif_rotate/no_exif.jpg", "tmpout/TestWriteThumbnail/jpeg_no_exif.jpg", false)

	// Non JPEG - no exif
	tWriteThumbnail(t, media, "png.png", "tmpout/TestWriteThumbnail/png.jpg", false)

	// Video - only if video is supported
	if hasVideoThumbnailSupport() {
		tWriteThumbnail(t, media, "video.mp4", "tmpout/TestWriteThumbnail/video.jpg", false)

		// Test invalid
		tWriteThumbnail(t, media, "invalidvideo.mp4", "tmpout/TestWriteThumbnail/invalidvideo.jpg", true)
		// Check that error indication file is created
		assertFileExist(t, "", "tmpcache/TestWriteThumbnail/invalidvideo.thumb.err.txt")
	}

	// Non existing file
	tWriteThumbnail(t, media, "dont_exist.jpg", "tmpout/TestWriteThumbnail/dont_exist.jpg", true)

	// Invalid file
	tWriteThumbnail(t, media, "invalid.jpg", "tmpout/TestWriteThumbnail/invalid.jpg", true)
	// Check that error indication file is created
	assertFileExist(t, "", "tmpcache/TestWriteThumbnail/invalid.thumb.err.txt")
	// Generate again - just for coverage
	tWriteThumbnail(t, media, "invalid.jpg", "tmpout/TestWriteThumbnail/invalid.jpg", true)

	// Disable thumb cache
	media = createMedia("testmedia", "tmpcache/TestWriteThumbnail", false, false, false, false, true, true, false, 0, false, false, false, false)

	// JPEG with embedded EXIF
	tWriteThumbnail(t, media, "jpeg.jpg", "tmpout/TestWriteThumbnail/jpeg.jpg", false)

	// Non JPEG - no exif
	tWriteThumbnail(t, media, "png.png", "tmpout/TestWriteThumbnail/png.jpg", true)
}

func TestVideoThumbnailSupport(t *testing.T) {
	// Since we cannot guarantee that ffmpeg is available on the test
	// host we will replace the ffmpeg command temporary
	origCmd := ffmpegCmd
	defer func() {
		ffmpegCmd = origCmd
	}()

	t.Logf("ffmpeg supported: %v", hasVideoThumbnailSupport())

	ffmpegCmd = "thiscommanddontexit"
	assertFalse(t, ffmpegCmd, hasVideoThumbnailSupport())

	ffmpegCmd = "cmd"
	shallBeTrueOnWindows := hasVideoThumbnailSupport()

	ffmpegCmd = "echo"
	shallBeTrueOnNonWindows := hasVideoThumbnailSupport()

	assertTrue(t, "Shall be true on at least one platform", shallBeTrueOnWindows || shallBeTrueOnNonWindows)
}

func tGenerateVideoThumbnail(t *testing.T, media *Media, inFileName, outFileName string) {
	t.Helper()
	os.Remove(outFileName)
	RestartTimer()
	err := media.cache.generateVideoThumbnail(inFileName, outFileName)
	LogTime(t, inFileName+"thumbnail generation: ")
	assertExpectNoErr(t, "", err)
	assertFileExist(t, "", outFileName)
	t.Logf("Manually check that %s thumbnail is ok", outFileName)
}

func TestGenerateVideoThumbnail(t *testing.T) {
	media := createMedia("testmedia", ".", true, false, false, false, true, true, false, 0, false, false, false, false)
	if !hasVideoThumbnailSupport() {
		t.Skip("ffmpeg not installed skipping test")
		return
	}
	tmp := "tmpout/TestGenerateVideoThumbnail"
	os.MkdirAll(tmp, os.ModePerm) // If already exist no problem
	tmpSpace := "tmpout/TestGenerateVideoThumbnail/with space in path"
	os.MkdirAll(tmpSpace, os.ModePerm) // If already exist no problem

	tGenerateVideoThumbnail(t, media, "testmedia/video.mp4", tmp+"/video_thumbnail.jpg")
	tGenerateVideoThumbnail(t, media, "testmedia/video.mp4", tmpSpace+"/video_thumbnail.jpg")

	// Test some invalid
	err := media.cache.generateVideoThumbnail("nonexisting.mp4", tmp+"dont_matter.jpg")
	assertExpectErr(t, "", err)
	err = media.cache.generateVideoThumbnail("invalidvideo.mp4", tmp+"/invalidvideo.jpg")
	assertExpectErr(t, "", err)
}

func TestGenerateThumbnails(t *testing.T) {
	cache := "tmpcache/TestGenerateThumbnails"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, false, false, true, true, false, 0, false, false, false, false)
	stat := media.generateCache("", true, true, false)
	assertEqualsInt(t, "", 1, stat.NbrOfFolders)
	assertEqualsInt(t, "", 20, stat.NbrOfImages)
	assertEqualsInt(t, "", 2, stat.NbrOfVideos)
	assertEqualsInt(t, "", 10, stat.NbrOfExif)
	assertEqualsInt(t, "", 9, stat.NbrOfImageThumb)
	assertEqualsInt(t, "", 0, stat.NbrOfImagePreview)
	assertEqualsInt(t, "", 0, stat.NbrOfFailedFolders)
	assertEqualsInt(t, "", 1, stat.NbrOfFailedImageThumb)
	assertEqualsInt(t, "", 0, stat.NbrOfFailedImagePreview)
	assertEqualsInt(t, "", 0, stat.NbrOfSmallImages)
	assertEqualsInt(t, "", 0, stat.NbrRemovedCacheFiles)
	if hasVideoThumbnailSupport() {
		assertEqualsInt(t, "", 1, stat.NbrOfVideoThumb)
		assertEqualsInt(t, "", 1, stat.NbrOfFailedVideoThumb)
		assertFileExist(t, "", filepath.Join(cache, "video.thumb.jpg"))
	} else {
		assertEqualsInt(t, "", 0, stat.NbrOfVideoThumb)
		assertEqualsInt(t, "", 2, stat.NbrOfFailedVideoThumb)
	}

	// Check that thumbnails where generated
	assertFileExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))

	assertCacheThumbExists(t, media.cache, "", "png.jpg")
	assertCacheThumbExists(t, media.cache, "", "gif.gif")
	assertCacheThumbExists(t, media.cache, "", "tiff.tiff")
	assertCacheThumbExists(t, media.cache, "", "exif_rotate/no_exif.jpg")

	// Check that thumbnails where not generated for EXIF images
	assertFileNotExist(t, "", filepath.Join(cache, "jpeg.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "jpeg_rotated.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "180deg.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "mirror.thumb.jpg"))
}

func TestGeneratePreviews(t *testing.T) {
	cache := "tmpcache/TestGeneratePreviews"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	// Create some unnecessary files that should be removed since enableCacheCleanup is true.
	unnecessaryFile := filepath.Join(cache, "unnecessary_cache_file.jpg")
	os.Create(unnecessaryFile)
	unnecessaryDirectory := filepath.Join(cache, "unnecessary_directory")
	os.MkdirAll(unnecessaryDirectory, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, false, false, true, true, true, 1280, false, false, false, true)
	stat := media.generateCache("", true, false, true)
	assertEqualsInt(t, "", 1, stat.NbrOfFolders)
	assertEqualsInt(t, "", 20, stat.NbrOfImages)
	assertEqualsInt(t, "", 2, stat.NbrOfVideos)
	assertEqualsInt(t, "", 10, stat.NbrOfExif)
	assertEqualsInt(t, "", 0, stat.NbrOfImageThumb)
	assertEqualsInt(t, "", 12, stat.NbrOfImagePreview)
	assertEqualsInt(t, "", 0, stat.NbrOfFailedFolders)
	assertEqualsInt(t, "", 0, stat.NbrOfFailedImageThumb)
	assertEqualsInt(t, "", 1, stat.NbrOfFailedImagePreview)
	assertEqualsInt(t, "", 7, stat.NbrOfSmallImages)
	assertEqualsInt(t, "", 2, stat.NbrRemovedCacheFiles)

	// Check that previews where generated
	assertFileExist(t, "", filepath.Join(cache, "png.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "normal.preview.jpg"))

	assertCachePreviewExists(t, media.cache, "", "png.jpg")
	assertCachePreviewExists(t, media.cache, "", "gif.gif")
	assertCachePreviewExists(t, media.cache, "", "exif_rotate/normal.jpg")

	// Check that no thumbnails where generated
	assertFileNotExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))

	// Check that unnecessary files are removed
	assertFileNotExist(t, "", unnecessaryFile)
	assertFileNotExist(t, "", unnecessaryDirectory)

}

func TestGenerateThumbnailsAndPreviews(t *testing.T) {
	cache := "tmpcache/TestGenerateThumbnailsAndPreviews"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	// Create some unnecessary files that should be kept since enableCacheCleanup is false.
	unnecessaryFile := filepath.Join(cache, "unnecessary_cache_file.jpg")
	os.Create(unnecessaryFile)
	unnecessaryDirectory := filepath.Join(cache, "unnecessary_directory")
	os.MkdirAll(unnecessaryDirectory, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, false, false, true, true, true, 1280, false, false, false, false)
	stat := media.generateCache("", true, true, true)
	assertEqualsInt(t, "", 1, stat.NbrOfFolders)
	assertEqualsInt(t, "", 20, stat.NbrOfImages)
	assertEqualsInt(t, "", 2, stat.NbrOfVideos)
	assertEqualsInt(t, "", 10, stat.NbrOfExif)
	assertEqualsInt(t, "", 9, stat.NbrOfImageThumb)
	assertEqualsInt(t, "", 12, stat.NbrOfImagePreview)
	assertEqualsInt(t, "", 0, stat.NbrOfFailedFolders)
	assertEqualsInt(t, "", 1, stat.NbrOfFailedImageThumb)
	assertEqualsInt(t, "", 1, stat.NbrOfFailedImagePreview)
	assertEqualsInt(t, "", 7, stat.NbrOfSmallImages)
	assertEqualsInt(t, "", 0, stat.NbrRemovedCacheFiles)

	// Check that previews where generated
	assertFileExist(t, "", filepath.Join(cache, "png.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "normal.preview.jpg"))

	// Check that thumbnails where generated
	assertFileExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))

	// Check that unnecessary files are kept
	assertFileExist(t, "", unnecessaryFile)
	assertFileExist(t, "", unnecessaryDirectory)

	//

}

func TestGenerateAllThumbnails(t *testing.T) {
	cache := "tmpcache/TestGenerateAllThumbnails"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, true, false, true, true, false, 0, false, false, false, false)

	for i := 0; i < 300; i++ {
		time.Sleep(100 * time.Millisecond)
		if !media.isPreCacheInProgress() {
			break
		}
	}
	assertFalse(t, "", media.isPreCacheInProgress())

	// Check that thumbnails where generated
	assertFileExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))

	// Check that thumbnails where not generated for EXIF images
	assertFileNotExist(t, "", filepath.Join(cache, "jpeg.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "jpeg_rotated.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "180deg.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "mirror.thumb.jpg"))

	if hasVideoThumbnailSupport() {
		assertFileExist(t, "", filepath.Join(cache, "video.thumb.jpg"))
	}
}

func TestGenerateAllPreviews(t *testing.T) {
	cache := "tmpcache/TestGenerateAllPreviews"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, false, false, true, true, true, 1280, false, true, false, false)

	for i := 0; i < 300; i++ {
		time.Sleep(100 * time.Millisecond)
		if !media.isPreCacheInProgress() {
			break
		}
	}
	assertFalse(t, "", media.isPreCacheInProgress())

	// Check that previews where generated
	assertFileExist(t, "", filepath.Join(cache, "png.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "normal.preview.jpg"))

	// Check that no previews where generated for "small" images
	assertFileNotExist(t, "", filepath.Join(cache, "tiff.preview.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "screenshot_viewer.preview.jpg"))

	// Check that no thumbnails where generated
	assertFileNotExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "video.thumb.jpg"))
}

func TestGenerateAllThumbsAndPreviews(t *testing.T) {
	cache := "tmpcache/TestGenerateAllThumbsAndPreviews"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, true, false, true, true, true, 1280, false, true, false, false)

	for i := 0; i < 300; i++ {
		time.Sleep(100 * time.Millisecond)
		if !media.isPreCacheInProgress() {
			break
		}
	}
	assertFalse(t, "", media.isPreCacheInProgress())

	// Check that thumbnails where generated
	assertFileExist(t, "", filepath.Join(cache, "png.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "tiff.thumb.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "no_exif.thumb.jpg"))

	// Check that previews where generated
	assertFileExist(t, "", filepath.Join(cache, "png.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "gif.preview.jpg"))
	assertFileExist(t, "", filepath.Join(cache, "exif_rotate", "normal.preview.jpg"))

}

func TestGenerateNoThumbnails(t *testing.T) {
	cache := "tmpcache/TestGenerateNoThumbnails"
	os.RemoveAll(cache)
	os.MkdirAll(cache, os.ModePerm)

	media := createMedia("testmedia", cache, true, false, false, false, true, true, false, 0, false, false, false, false)

	assertFalse(t, "", media.isPreCacheInProgress())
	time.Sleep(100 * time.Millisecond)
	assertFalse(t, "", media.isPreCacheInProgress())

	// Check that no thumbnails where generated
	assertFileNotExist(t, "", filepath.Join(cache, "_png.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "_gif.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "_tiff.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "_video.jpg"))
	assertFileNotExist(t, "", filepath.Join(cache, "exif_rotate", "_no_exif.jpg"))

}

func TestGetImageWidthAndHeight(t *testing.T) {
	media := createMedia("", "", true, false, false, false, true, true, false, 0, false, false, false, false)

	width, height, err := media.getImageWidthAndHeight("testmedia/jpeg.jpg")
	assertExpectNoErr(t, "", err)
	assertEqualsInt(t, "image width", 4128, width)
	assertEqualsInt(t, "image height", 2322, height)

	width, height, err = media.getImageWidthAndHeight("testmedia/gif.gif")
	assertExpectNoErr(t, "", err)
	assertEqualsInt(t, "image width", 3264, width)
	assertEqualsInt(t, "image height", 2448, height)

	width, height, err = media.getImageWidthAndHeight("testmedia/png.png")
	assertExpectNoErr(t, "", err)
	assertEqualsInt(t, "image width", 1632, width)
	assertEqualsInt(t, "image height", 1224, height)

	width, height, err = media.getImageWidthAndHeight("testmedia/tiff.tiff")
	assertExpectNoErr(t, "", err)
	assertEqualsInt(t, "image width", 979, width)
	assertEqualsInt(t, "image height", 734, height)

	// Test invalid
	_, _, err = media.getImageWidthAndHeight("testmedia/invalid.jpg")
	assertExpectErr(t, "", err)
}

func TestPreviewPath(t *testing.T) {
	media := createMedia("/c/mediapath", "/d/thumbpath", true, false, false, false, true, true, true, 1280, false, false, false, false)

	previewPath, err := media.cache.previewPath("myimage.jpg")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/myimage.preview.jpg", previewPath)

	previewPath, err = media.cache.previewPath("subdrive/myimage.jpg")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/subdrive/myimage.preview.jpg", previewPath)

	previewPath, err = media.cache.previewPath("subdrive/myimage.png")
	assertExpectNoErr(t, "", err)
	assertEqualsStr(t, "", "/d/thumbpath/subdrive/myimage.preview.jpg", previewPath)

	_, err = media.cache.previewPath("subdrive/myimage")
	assertExpectErr(t, "", err)

	_, err = media.cache.previewPath("subdrive/../../hacker")
	assertExpectErr(t, "", err)
}

func tGenerateImagePreview(t *testing.T, media *Media, inFileName, outFileName string) {
	t.Helper()
	os.Remove(outFileName)
	RestartTimer()
	err := media.cache.generateImagePreview(inFileName, outFileName)
	LogTime(t, inFileName+" preview generation: ")
	assertExpectNoErr(t, "", err)
	assertFileExist(t, "", outFileName)
	// Check dimensions
	width, height, err := media.getImageWidthAndHeight(outFileName)
	assertExpectNoErr(t, "reading dimensions", err)
	assertFalse(t, "preview width", width > media.cache.previewMaxSide)
	assertFalse(t, "preview height", height > media.cache.previewMaxSide)
}

func TestGenerateImagePreview(t *testing.T) {
	os.MkdirAll("tmpout/TestGenerateImagePreview", os.ModePerm) // If already exist no problem

	media := createMedia("", "", true, false, false, false, true, true, true, 1280, false, false, false, false)

	tGenerateImagePreview(t, media, "testmedia/jpeg.jpg", "tmpout/TestGenerateImagePreview/jpeg_preview.jpg")
	tGenerateImagePreview(t, media, "testmedia/jpeg_rotated.jpg", "tmpout/TestGenerateImagePreview/jpeg_rotated_preview.jpg")
	tGenerateImagePreview(t, media, "testmedia/png.png", "tmpout/TestGenerateImagePreview/png_preview.jpg")
	tGenerateImagePreview(t, media, "testmedia/gif.gif", "tmpout/TestGenerateImagePreview/gif_preview.jpg")
	tGenerateImagePreview(t, media, "testmedia/tiff.tiff", "tmpout/TestGenerateImagePreview/tiff_preview.jpg")
	tGenerateImagePreview(t, media, "testmedia/exif_rotate/no_exif.jpg", "tmpout/TestGenerateImagePreview/exif_rotate/no_exif_preview.jpg")

	// Test some invalid
	err := media.cache.generateImagePreview("nonexisting.png", "dont_matter.png")
	assertExpectErr(t, "", err)

	err = media.cache.generateImagePreview("testmedia/invalid.jpg", "dont_matter.jpg")
	assertExpectErr(t, "", err)
}

func tWritePreview(t *testing.T, media *Media, inFileName, outFileName string, failExpected bool) {
	t.Helper()
	os.Remove(outFileName)
	outFile, err := os.Create(outFileName)
	assertExpectNoErr(t, "unable to create out", err)
	defer outFile.Close()
	err = media.writePreview(outFile, inFileName)
	if failExpected {
		assertExpectErr(t, "should fail", err)
	} else {
		assertExpectNoErr(t, "unable to write preview", err)
		// Check dimensions
		width, height, err := media.getImageWidthAndHeight(outFileName)
		assertExpectNoErr(t, "reading dimensions", err)
		assertFalse(t, "preview width", width > media.cache.previewMaxSide)
		assertFalse(t, "preview height", height > media.cache.previewMaxSide)
	}
}

func TestWritePreview(t *testing.T) {
	os.RemoveAll("tmpcache/TestWritePreview")
	os.MkdirAll("tmpcache/TestWritePreview", os.ModePerm)
	os.RemoveAll("tmpout/TestWritePreview")
	os.MkdirAll("tmpout/TestWritePreview", os.ModePerm)

	media := createMedia("testmedia", "tmpcache/TestWritePreview", true, false, false, false, true, true, true, 970, false, false, false, false)

	// JPEG
	tWritePreview(t, media, "jpeg.jpg", "tmpout/TestWritePreview/jpeg.jpg", false)

	// Same file again, get cached result
	tWritePreview(t, media, "jpeg.jpg", "tmpout/TestWritePreview/jpeg.jpg", false)

	// PNG
	tWritePreview(t, media, "png.png", "tmpout/TestWritePreview/png.jpg", false)

	// TIFF
	tWritePreview(t, media, "tiff.tiff", "tmpout/TestWritePreview/tiff.tiff", false)

	// Video - should fail
	tWritePreview(t, media, "video.mp4", "tmpout/TestWritePreview/video.jpg", true)

	// Image smaller than previewMaxSide should fail
	tWritePreview(t, media, "screenshot_browser.jpg", "tmpout/TestWritePreview/screenshot_browser.jpg", true)

	// Non existing file
	tWritePreview(t, media, "dont_exist.jpg", "tmpout/TestWritePreview/dont_exist.jpg", true)

	// Invalid file
	tWritePreview(t, media, "invalid.jpg", "tmpout/TestWritePreview/invalid.jpg", true)
	// Check that error indication file is created
	assertFileExist(t, "", "tmpcache/TestWritePreview/invalid.preview.err.txt")
	// Regenerate for increased coverage
	tWritePreview(t, media, "invalid.jpg", "tmpout/TestWritePreview/invalid.jpg", true)

	// Invalid path
	tWritePreview(t, media, "../../secret.jpg", "tmpout/TestWritePreview/invalid.jpg", true)

	// Disable preview
	media = createMedia("testmedia", "tmpcache/TestWritePreview", false, false, false, false, true, true, false, 0, false, false, false, false)

	// Should fail since preview is disabled now
	tWritePreview(t, media, "jpeg.jpg", "tmpout/TestWritePreview/jpeg.jpg", true)
}
