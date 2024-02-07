// Assertion helpers of golang unit tests
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"testing"
)

func assertTrue(t *testing.T, message string, check bool) {
	t.Helper()
	if !check {
		debug.PrintStack()
		t.Fatal(message)
	}
}

func assertFalse(t *testing.T, message string, check bool) {
	t.Helper()
	if check {
		debug.PrintStack()
		t.Fatal(message)
	}
}

func assertExpectNoErr(t *testing.T, message string, err error) {
	t.Helper()
	if err != nil {
		debug.PrintStack()
		t.Fatalf("%s : %s", message, err)
	}
}

func assertExpectErr(t *testing.T, message string, err error) {
	t.Helper()
	if err == nil {
		debug.PrintStack()
		t.Fatal(message)
	}
}

func assertEqualsInt(t *testing.T, message string, expected int, actual int) {
	t.Helper()
	assertTrue(t, fmt.Sprintf("%s\nExpected: %d, Actual: %d", message, expected, actual), expected == actual)
}

func assertEqualsStr(t *testing.T, message string, expected string, actual string) {
	t.Helper()
	assertTrue(t, fmt.Sprintf("%s\nExpected: %s, Actual: %s", message, expected, actual), expected == actual)
}

func assertEqualsBool(t *testing.T, message string, expected bool, actual bool) {
	t.Helper()
	assertTrue(t, fmt.Sprintf("%s\nExpected: %t, Actual: %t", message, expected, actual), expected == actual)
}

func assertFileExist(t *testing.T, message string, name string) {
	t.Helper()
	if _, err := os.Stat(name); err != nil {
		debug.PrintStack()
		t.Fatalf("%s : %s", message, err)
	}
}

func assertFileNotExist(t *testing.T, message string, name string) {
	t.Helper()
	if _, err := os.Stat(name); err == nil {
		debug.PrintStack()
		t.Fatalf("%s : %s exist but shall not", message, name)
	}
}

func assertCacheThumbExists(t *testing.T, c *Cache, message string, name string) {
	t.Helper()
	if !c.hasThumbnail(name) {
		debug.PrintStack()
		t.Fatalf("%s : %s thumb does not exist in cache", message, name)
	}
}

func assertCachePreviewExists(t *testing.T, c *Cache, message string, name string) {
	t.Helper()
	if !c.hasPreview(name) {
		debug.PrintStack()
		t.Fatalf("%s : %s preview does not exist in cache", message, name)
	}
}
