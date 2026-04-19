package imagegen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func ValidateGenerate(req GenerateRequest, outputPath string) error {
	if req.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if err := checkExtension(outputPath, req.OutputFormat); err != nil {
		return err
	}
	if !validEnum(req.OutputFormat, "png", "jpeg", "webp") {
		return fmt.Errorf("format must be one of png, jpeg, webp (got %q)", req.OutputFormat)
	}
	if !validEnum(req.Size, "1024x1024", "1536x1024", "1024x1536", "auto") {
		return fmt.Errorf("size must be one of 1024x1024, 1536x1024, 1024x1536, auto (got %q)", req.Size)
	}
	if !validEnum(req.Background, "transparent", "opaque", "auto") {
		return fmt.Errorf("background must be one of transparent, opaque, auto (got %q)", req.Background)
	}
	if !validEnum(req.Quality, "low", "medium", "high", "auto") {
		return fmt.Errorf("quality must be one of low, medium, high, auto (got %q)", req.Quality)
	}
	if req.Background == "transparent" && req.OutputFormat == "jpeg" {
		return fmt.Errorf("background=transparent is incompatible with format=jpeg")
	}
	if req.Background == "transparent" && (req.Quality == "low" || req.Quality == "auto") {
		return fmt.Errorf("background=transparent requires quality=medium or quality=high")
	}
	if req.N < 1 || req.N > 10 {
		return fmt.Errorf("n must be between 1 and 10 (got %d)", req.N)
	}
	return nil
}

func ValidateEdit(req EditRequest, outputPath string) error {
	if err := ValidateGenerate(req.GenerateRequest, outputPath); err != nil {
		return err
	}
	if err := checkInputFile("input_path", req.ImagePath); err != nil {
		return err
	}
	if req.MaskPath != "" {
		if err := checkInputFile("mask_path", req.MaskPath); err != nil {
			return err
		}
	}
	return nil
}

func checkExtension(outputPath, format string) error {
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch format {
	case "png":
		if ext != ".png" {
			return fmt.Errorf("output_path extension %q does not match format=png (expected .png)", ext)
		}
	case "jpeg":
		if ext != ".jpg" && ext != ".jpeg" {
			return fmt.Errorf("output_path extension %q does not match format=jpeg (expected .jpg or .jpeg)", ext)
		}
	case "webp":
		if ext != ".webp" {
			return fmt.Errorf("output_path extension %q does not match format=webp (expected .webp)", ext)
		}
	}
	return nil
}

func checkInputFile(name, p string) error {
	if !filepath.IsAbs(p) {
		return fmt.Errorf("%s must be an absolute path (got %q)", name, p)
	}
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("%s %q: %w", name, p, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular file", name, p)
	}
	return nil
}

func validEnum(v string, allowed ...string) bool {
	return slices.Contains(allowed, v)
}
