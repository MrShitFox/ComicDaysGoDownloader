package main

import (
	"image"
	"path/filepath"
	"testing"
)

func TestSaveImageRequiresDeobfuscate(t *testing.T) {
	processor := NewImageContext(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err := processor.SaveImage(filepath.Join(t.TempDir(), "out.png")); err == nil {
		t.Fatal("SaveImage succeeded before Deobfuscate")
	}
}
