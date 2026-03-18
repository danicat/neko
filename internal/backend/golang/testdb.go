package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const dbFile = "testquery.db"

func goBuildTestDB(ctx context.Context, dir string, pkg string) error {
	if pkg == "" {
		pkg = "./..."
	}

	dbPath := filepath.Join(dir, dbFile)

	buildCmd := exec.CommandContext(ctx, "go", "run", "github.com/danicat/testquery@latest",
		"build", "--pkg", pkg, "--output", dbFile)
	buildCmd.Dir = dir
	out, buildErr := buildCmd.CombinedOutput()

	if buildErr != nil {
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("failed to build test database: %v\n%s", buildErr, filterNoise(string(out)))
		}
	}

	return nil
}

func goQueryTestDB(ctx context.Context, dir string, query string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "run", "github.com/danicat/testquery@latest",
		"query", "--db", dbFile, "--format", "table", query)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	output := filterNoise(string(out))

	if err != nil && output == "" {
		return "", fmt.Errorf("test query failed: %v", err)
	}

	if err != nil {
		return fmt.Sprintf("⚠️ Query completed with warnings:\n\n%s", output), nil
	}

	if output == "" {
		return "Query returned no results.", nil
	}

	return output, nil
}
