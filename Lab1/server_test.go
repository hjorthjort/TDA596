package main

import (
	"testing"
)

func TestServeGoodPorts(t *testing.T) {
	goodPorts := [5]string{"0", "1", "8080", "65535", ""}
	for _, port := range goodPorts {
		_, err := getListener(port)
		if err != nil {
			t.Fatalf("Serve failed for port %s: %v", port, err)
		}
	}
}

func TestServeBadPorts(t *testing.T) {
	badPorts := [3]string{"-1", "65536", "abc"}
	for _, port := range badPorts {
		_, err := getListener(port)
		if err == nil {
			t.Fatalf("Serve succeeded for bad port %s", port)
		}
	}
}
