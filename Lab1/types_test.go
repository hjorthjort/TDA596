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
		if !isSupportedFileType(path) {
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
		if isSupportedFileType(path) {
			t.Errorf("Expected %s to not be a supported file type, but it was.", path)
		}
	}
}
