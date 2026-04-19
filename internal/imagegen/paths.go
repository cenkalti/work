package imagegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveOutputPath(p, defaultDir string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("output_path is required")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	if defaultDir == "" {
		return "", fmt.Errorf("output_path %q is relative and IMAGE_GEN_DEFAULT_DIR is not set", p)
	}
	if !filepath.IsAbs(defaultDir) {
		return "", fmt.Errorf("IMAGE_GEN_DEFAULT_DIR %q must be absolute", defaultDir)
	}
	return filepath.Clean(filepath.Join(defaultDir, p)), nil
}

func SuffixedPaths(abs string, n int) []string {
	if n <= 1 {
		return []string{abs}
	}
	ext := filepath.Ext(abs)
	stem := strings.TrimSuffix(abs, ext)
	out := make([]string, n)
	for i := range n {
		out[i] = fmt.Sprintf("%s-%d%s", stem, i+1, ext)
	}
	return out
}

func ensureParentDir(abs string) error {
	return os.MkdirAll(filepath.Dir(abs), 0o755)
}
