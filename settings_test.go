package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsDefault(t *testing.T) {
	contents :=
		`
port = 9834
mediapath = Y:\pictures`
	fullPath := createConfigFile(t, "TestSettingsDefault.conf", contents)
	s := loadSettings(fullPath)

	// Mandatory values
	assertEqualsInt(t, "port", 9834, s.port)
	assertEqualsStr(t, "mediaPath", "Y:\\pictures", s.mediaPath)

	// All default on optional
	assertEqualsStr(t, "cachePath", filepath.Join(os.TempDir(), "mediaweb"), s.cachePath)
	assertEqualsBool(t, "enablethumbCache", true, s.enableThumbCache)
	assertEqualsBool(t, "genthumbsonstartup", false, s.genThumbsOnStartup)
	assertEqualsBool(t, "genthumbsonadd", true, s.genThumbsOnAdd)
	assertEqualsBool(t, "autoRotate", true, s.autoRotate)
	assertEqualsBool(t, "enablepreview", false, s.enablePreview)
	assertEqualsInt(t, "previewmaxside", 1280, s.previewMaxSide)
	assertEqualsBool(t, "genpreviewonstartup", false, s.genPreviewOnStartup)
	assertEqualsBool(t, "genpreviewonadd", true, s.genPreviewOnAdd)
	assertEqualsBool(t, "enablecachecleanup", false, s.enableCacheCleanup)
	// assertEqualsInt(t, "logLevel", int(llog.LvlInfo), int(s.logLevel))
	assertEqualsStr(t, "logFile", "", s.logFile)
	assertEqualsStr(t, "userName", "", s.userName)
	assertEqualsStr(t, "password", "", s.password)
	assertEqualsStr(t, "ip", "", s.ip)
	assertEqualsStr(t, "tlsCertFile", "", s.tlsCertFile)
	assertEqualsStr(t, "tlsKeyFile", "", s.tlsKeyFile)

}

func TestSettings(t *testing.T) {
	contents :=
		`
port = 80
ip = 192.168.1.2
mediapath = /media/usb/pictures
cachepath = /tmp/thumb
enablethumbcache = off
genthumbsonstartup = on
genthumbsonadd = off
autorotate = false
enablepreview = true
previewmaxside = 1920
genpreviewonstartup = on
genpreviewonadd = off
enablecachecleanup = on
loglevel = debug
logfile = /tmp/log/mediaweb.log
username = an_email@password.com
password = """A!#_q7*+"""
tlscertfile = /file/my_cert_file.crt
tlskeyfile = /file/my_cert_file.key
`
	fullPath := createConfigFile(t, "TestSettings.conf", contents)
	s := loadSettings(fullPath)

	// Mandatory values
	assertEqualsInt(t, "port", 80, s.port)
	assertEqualsStr(t, "mediaPath", "/media/usb/pictures", s.mediaPath)

	// Check set values on optional
	assertEqualsStr(t, "cachePath", "/tmp/thumb", s.cachePath)
	assertEqualsBool(t, "enableThumbCache", false, s.enableThumbCache)
	assertEqualsBool(t, "genthumbsonstartup", true, s.genThumbsOnStartup)
	assertEqualsBool(t, "genthumbsonadd", false, s.genThumbsOnAdd)
	assertEqualsBool(t, "autoRotate", false, s.autoRotate)
	assertEqualsBool(t, "enablepreview", true, s.enablePreview)
	assertEqualsInt(t, "previewmaxside", 1920, s.previewMaxSide)
	assertEqualsBool(t, "genpreviewonstartup", true, s.genPreviewOnStartup)
	assertEqualsBool(t, "genpreviewonadd", false, s.genPreviewOnAdd)
	assertEqualsBool(t, "enablecachecleanup", true, s.enableCacheCleanup)
	// assertEqualsInt(t, "logLevel", int(llog.LvlDebug), int(s.logLevel))
	assertEqualsStr(t, "logFile", "/tmp/log/mediaweb.log", s.logFile)
	assertEqualsStr(t, "userName", "an_email@password.com", s.userName)
	assertEqualsStr(t, "password", "A!#_q7*+", s.password)
	assertEqualsStr(t, "ip", "192.168.1.2", s.ip)
	assertEqualsStr(t, "tlsCertFile", "/file/my_cert_file.crt", s.tlsCertFile)
	assertEqualsStr(t, "tlsKeyFile", "/file/my_cert_file.key", s.tlsKeyFile)

}

func TestSettingsInvalidOptional(t *testing.T) {
	contents :=
		`
port = 80
mediapath = /media/usb/pictures
cachepath = /tmp/thumb
enablethumbcache = 33
genthumbsonstartup = -1
genthumbsonadd = 5.5
autorotate = invalid
enablepreview = 27
previewmaxside = invalid
enablethumbcache = -6
genthumbsonstartup = 67
enablecachecleanup = 4.5
loglevel = debug
logfile = /tmp/log/mediaweb.log
`
	fullPath := createConfigFile(t, "TestSettings.conf", contents)
	s := loadSettings(fullPath)

	// Mandatory values
	assertEqualsInt(t, "port", 80, s.port)
	assertEqualsStr(t, "mediaPath", "/media/usb/pictures", s.mediaPath)

	// Check set values on optional
	assertEqualsStr(t, "cachePath", "/tmp/thumb", s.cachePath)
	assertEqualsInt(t, "previewmaxside", 1280, s.previewMaxSide)
	// assertEqualsInt(t, "logLevel", int(llog.LvlDebug), int(s.logLevel))
	assertEqualsStr(t, "logFile", "/tmp/log/mediaweb.log", s.logFile)

	// Should be default on invalid values
	assertEqualsBool(t, "enablethumbCache", true, s.enableThumbCache)
	assertEqualsBool(t, "genthumbsonstartup", false, s.genThumbsOnStartup)
	assertEqualsBool(t, "genthumbsonadd", true, s.genThumbsOnAdd)
	assertEqualsBool(t, "autoRotate", true, s.autoRotate)
	assertEqualsBool(t, "enablepreview", false, s.enablePreview)
	assertEqualsBool(t, "genpreviewonstartup", false, s.genPreviewOnStartup)
	assertEqualsBool(t, "genpreviewonadd", true, s.genPreviewOnAdd)
	assertEqualsBool(t, "enablecachecleanup", false, s.enableCacheCleanup)

}

func TestSettingsBackwardsCompatibility(t *testing.T) {
	contents :=
		`
port = 80
mediapath = /media/usb/pictures
thumbpath = /tmp/thumb
loglevel = debug
logfile = /tmp/log/mediaweb.log
`
	fullPath := createConfigFile(t, "TestSettings.conf", contents)
	s := loadSettings(fullPath)

	// Mandatory values
	assertEqualsInt(t, "port", 80, s.port)
	assertEqualsStr(t, "mediaPath", "/media/usb/pictures", s.mediaPath)

	// Check that cachepath is working with thumbpath
	assertEqualsStr(t, "cachePath", "/tmp/thumb", s.cachePath)

}

func expectPanic(t *testing.T) {
	// Panic handler (panic is expected)
	recover()
	confPaths = defaultConfPaths // Reset default configuration paths
	t.Log("No worry. Panic is expected in the test!!")
}

func TestSettingsNotExisting(t *testing.T) {
	defer expectPanic(t)
	loadSettings("dontexist.conf")
	t.Fatal("Non existing file. Panic expected")
}

func TestSettingsMissingPort(t *testing.T) {
	contents :=
		`
mediapath = Y:\pictures`
	fullPath := createConfigFile(t, "TestSettingsMissingPort.conf", contents)
	defer expectPanic(t)
	loadSettings(fullPath)
	t.Fatal("Panic expected")
}

func TestSettingsInvalidPort(t *testing.T) {
	contents :=
		`port=nonint
mediapath = Y:\pictures`
	fullPath := createConfigFile(t, "TestSettingsInvalidPort.conf", contents)
	defer expectPanic(t)
	loadSettings(fullPath)
	t.Fatal("Panic expected")
}

func TestSettingsMissingMediaPath(t *testing.T) {
	contents :=
		`port=80`
	fullPath := createConfigFile(t, "TestSettingsMissingMediaPath.conf", contents)
	defer expectPanic(t)
	loadSettings(fullPath)
	t.Fatal("Panic expected")
}

func TestToLogLvl(t *testing.T) {
	// checkLvl(t, llog.LvlTrace, "trace")
	// checkLvl(t, llog.LvlDebug, "debug")
	// checkLvl(t, llog.LvlInfo, "info")
	// checkLvl(t, llog.LvlWarn, "warn")
	// checkLvl(t, llog.LvlError, "error")
	// checkLvl(t, llog.LvlPanic, "panic")

	// // Invalid shall be info
	// checkLvl(t, llog.LvlInfo, "")
	// checkLvl(t, llog.LvlInfo, "invalid")

}

// func checkLvl(t *testing.T, expected llog.Level, strLevel string) {
// 	level := toLogLvl(strLevel)
// 	if level != expected {
// 		t.Fatalf("%s should be level %d but was %d", strLevel, int(expected), int(level))
// 	}

// }

// createConfigFile creates a configuration file. Returns the full path to it.
func createConfigFile(t *testing.T, name, contents string) string {
	os.MkdirAll("tmpout", os.ModePerm)
	fullName := "tmpout/" + name
	os.Remove(fullName) // Remove old if it exist
	err := os.WriteFile(fullName, []byte(contents), 0644)
	if err != nil {
		t.Fatalf("Unable to create configuration file. Reason: %s", err)
	}
	return fullName
}

func TestFindConfFile(t *testing.T) {
	// Default
	path := findConfFile()
	if path != "mediaweb.conf" {
		t.Fatalf("It should have found mediaweb.conf but found %s", path)
	}
}

func TestFindConfFileMissing(t *testing.T) {
	defer expectPanic(t)

	confPaths = []string{"dontexist.conf", "/etc/dontexist.conf"}

	findConfFile() // Shall panic
	t.Fatalf("Should have paniced here")
}

func TestPathEquals(t *testing.T) {
	assertTrue(t, "", pathEquals("adir", "adir"))
	assertTrue(t, "", pathEquals("adir/anotherdir", "adir/anotherdir"))
	assertTrue(t, "", pathEquals("adir/anotherdir", "adir/anotherdir/third/.."))

	assertFalse(t, "", pathEquals("adir", "bdir"))
	assertFalse(t, "", pathEquals("sameroot/leaf1", "sameroot/leaf2"))
	assertFalse(t, "", pathEquals("root1/leaf1", "root2/leaf1"))
	assertFalse(t, "", pathEquals("/unix/u", "C:\\windows\\w"))
}

func TestSettingsSameMediaAndCachePath(t *testing.T) {
	contents :=
		`
port = 80
mediapath = Y:\pictures
cachepath = Y:\pictures`
	fullPath := createConfigFile(t, "TestSettingsSameMediaAndCachePath.conf", contents)
	defer expectPanic(t)
	loadSettings(fullPath)
	t.Fatal("Panic expected")
}
