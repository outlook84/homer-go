package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveContained(root, path string) (string, string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	target := filepath.Clean(path)
	if !filepath.IsAbs(target) {
		target = filepath.Join(rootAbs, target)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}

	realRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", "", err
	}
	realRoot = filepath.Clean(realRoot)
	realTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", "", err
	}
	realTarget = filepath.Clean(realTarget)

	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("path must stay under root directory")
	}
	return realTarget, rel, nil
}
