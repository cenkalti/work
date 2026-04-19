package imagegen

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveOutputPath_Absolute(t *testing.T) {
	got, err := ResolveOutputPath("/tmp/foo.png", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/foo.png" {
		t.Fatalf("got %q, want /tmp/foo.png", got)
	}
}

func TestResolveOutputPath_RelativeWithDefault(t *testing.T) {
	got, err := ResolveOutputPath("foo.png", "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Clean("/tmp/foo.png")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveOutputPath_RelativeWithoutDefault(t *testing.T) {
	if _, err := ResolveOutputPath("foo.png", ""); err == nil {
		t.Fatal("expected error for relative path without default dir")
	}
}

func TestResolveOutputPath_RelativeDefault(t *testing.T) {
	if _, err := ResolveOutputPath("foo.png", "relative/dir"); err == nil {
		t.Fatal("expected error for relative default dir")
	}
}

func TestResolveOutputPath_Empty(t *testing.T) {
	if _, err := ResolveOutputPath("", "/tmp"); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestSuffixedPaths_One(t *testing.T) {
	got := SuffixedPaths("/tmp/foo.png", 1)
	want := []string{"/tmp/foo.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSuffixedPaths_Many(t *testing.T) {
	got := SuffixedPaths("/tmp/foo.png", 3)
	want := []string{"/tmp/foo-1.png", "/tmp/foo-2.png", "/tmp/foo-3.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSuffixedPaths_HiddenFile(t *testing.T) {
	// Go's filepath.Ext treats ".foo.png" as having extension ".png",
	// so the suffix is inserted before .png.
	got := SuffixedPaths("/tmp/.foo.png", 2)
	want := []string{"/tmp/.foo-1.png", "/tmp/.foo-2.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSuffixedPaths_ZeroOrNegative(t *testing.T) {
	got := SuffixedPaths("/tmp/foo.png", 0)
	if !reflect.DeepEqual(got, []string{"/tmp/foo.png"}) {
		t.Fatalf("n=0: got %v", got)
	}
	got = SuffixedPaths("/tmp/foo.png", -3)
	if !reflect.DeepEqual(got, []string{"/tmp/foo.png"}) {
		t.Fatalf("n<0: got %v", got)
	}
}
