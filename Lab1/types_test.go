package main

import "testing"

func TestIsSupportedFileType(t *testing.T) {
	//html, txt, gif, jpeg, jpg, or css
	// valid paths with differnet lengths and splits
	testValidPaths := [12]string{
		"index.html",
		"/example///foo/bar.html",
		"file.txt",
		"/path/to/file.txt",
		"image.gif",
		"/images/image.gif",
		"photo.jpeg",
		"/photos/photo.jpeg",
		"picture.jpg",
		"/pictures/picture.jpg",
		"style.css",
		"/styles/style.css",
	}
	for _, path := range testValidPaths {
		if !isSupportedPath(path, "GET") {
			t.Errorf("Expected %s to be a supported file type, but it was not.", path)
		}
	}
}

func TestIsNotSupportedFileType(t *testing.T) {
	testInvalidPaths := [6]string{
		"index.php",
		"/example///foo/bar.php",
		"file.md",
		"/path/to/file.md",
		"image.bmp",
		"/images/image.bmp",
	}
	for _, path := range testInvalidPaths {
		if isSupportedPath(path, "GET") {
			t.Errorf("Expected %s to not be a supported file type, but it was.", path)
		}
	}
}

func TestIsSupportedPubSubPath(t *testing.T) {
	testValidPaths := map[string]string{
		"/subscribe":   "POST",
		"/unsubscribe": "POST",
		"/publish":     "POST",
		"/events":      "GET",
	}
	for path, method := range testValidPaths {
		if !isSupportedPath(path, method) {
			t.Errorf("incorrectly unsupported path %s, method: %s", path,
				method)
		}
	}
}

func TestIsNotSupportedPubSubPath(t *testing.T) {
	testInvalidPaths := map[string]string{
		"/foobar":      "DELETE",
		"/barbaz":      "PUT",
		"/bazboo":      "POST",
		"/beebop":      "GET",
		"/subscribe":   "GET",
		"/unsubscribe": "GET",
		"/publish":     "GET",
		"/events":      "POST",
	}
	for path, method := range testInvalidPaths {
		if isSupportedPath(path, method) {
			t.Errorf("incorrectly supported. path: %s; method: %s.", path,
				method)
		}
	}
}
