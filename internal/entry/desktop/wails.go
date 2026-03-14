//go:build wails

package desktop

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"pause/internal/app"
)

func RunWailsFromEmbedded(configPath string, embedded embed.FS, subdir string) error {
	assets, err := fs.Sub(embedded, subdir)
	if err != nil {
		return fmt.Errorf("load embedded frontend assets: %w", err)
	}
	return app.RunWails(configPath, assets)
}

func RunWailsFromDirCandidates(configPath string, candidates ...string) error {
	assets, err := dirFSFromCandidates(candidates...)
	if err != nil {
		return err
	}
	return app.RunWails(configPath, assets)
}

func dirFSFromCandidates(candidates ...string) (fs.FS, error) {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return os.DirFS(candidate), nil
		}
	}
	return nil, fmt.Errorf("frontend dist not found in candidates: %v", candidates)
}
