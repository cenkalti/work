package imagegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baseReq() GenerateRequest {
	return GenerateRequest{
		Prompt:       "a fox",
		Size:         "1024x1024",
		Background:   "auto",
		OutputFormat: "png",
		Quality:      "high",
		N:            1,
	}
}

func TestValidateGenerate_Happy(t *testing.T) {
	if err := ValidateGenerate(baseReq(), "/tmp/foo.png"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGenerate_JpegAcceptsBothExtensions(t *testing.T) {
	r := baseReq()
	r.OutputFormat = "jpeg"
	r.Background = "auto"
	for _, p := range []string{"/tmp/foo.jpg", "/tmp/foo.jpeg", "/tmp/FOO.JPG"} {
		if err := ValidateGenerate(r, p); err != nil {
			t.Errorf("path %q: unexpected error: %v", p, err)
		}
	}
}

func TestValidateGenerate_ExtensionMismatch(t *testing.T) {
	if err := ValidateGenerate(baseReq(), "/tmp/foo.jpg"); err == nil {
		t.Fatal("expected error for .jpg with format=png")
	}
}

func TestValidateGenerate_TransparentJpeg(t *testing.T) {
	r := baseReq()
	r.OutputFormat = "jpeg"
	r.Background = "transparent"
	err := ValidateGenerate(r, "/tmp/foo.jpg")
	if err == nil {
		t.Fatal("expected error for transparent+jpeg")
	}
	if !strings.Contains(err.Error(), "transparent") {
		t.Errorf("error should mention transparency: %v", err)
	}
}

func TestValidateGenerate_TransparentLowQuality(t *testing.T) {
	r := baseReq()
	r.Background = "transparent"
	r.Quality = "low"
	if err := ValidateGenerate(r, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for transparent+low")
	}
	r.Quality = "auto"
	if err := ValidateGenerate(r, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for transparent+auto")
	}
}

func TestValidateGenerate_TransparentMediumHighOK(t *testing.T) {
	r := baseReq()
	r.Background = "transparent"
	for _, q := range []string{"medium", "high"} {
		r.Quality = q
		if err := ValidateGenerate(r, "/tmp/foo.png"); err != nil {
			t.Errorf("quality=%s: unexpected error: %v", q, err)
		}
	}
}

func TestValidateGenerate_NOutOfRange(t *testing.T) {
	r := baseReq()
	for _, n := range []int{0, -1, 11, 100} {
		r.N = n
		if err := ValidateGenerate(r, "/tmp/foo.png"); err == nil {
			t.Errorf("n=%d: expected error", n)
		}
	}
}

func TestValidateGenerate_BadEnum(t *testing.T) {
	r := baseReq()
	r.Size = "2048x2048"
	if err := ValidateGenerate(r, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for unknown size")
	}
}

func TestValidateEdit_InputPathMissing(t *testing.T) {
	req := EditRequest{GenerateRequest: baseReq(), ImagePath: "/nonexistent/path/xyz.png"}
	if err := ValidateEdit(req, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for missing input_path")
	}
}

func TestValidateEdit_InputPathRelative(t *testing.T) {
	req := EditRequest{GenerateRequest: baseReq(), ImagePath: "rel.png"}
	if err := ValidateEdit(req, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for relative input_path")
	}
}

func TestValidateEdit_Happy(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "in.png")
	if err := os.WriteFile(inputPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}
	req := EditRequest{GenerateRequest: baseReq(), ImagePath: inputPath}
	if err := ValidateEdit(req, "/tmp/foo.png"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEdit_MaskOptional(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "in.png")
	if err := os.WriteFile(inputPath, []byte{0x89}, 0o644); err != nil {
		t.Fatal(err)
	}
	req := EditRequest{GenerateRequest: baseReq(), ImagePath: inputPath, MaskPath: "/nonexistent/mask.png"}
	if err := ValidateEdit(req, "/tmp/foo.png"); err == nil {
		t.Fatal("expected error for missing mask_path")
	}
}
