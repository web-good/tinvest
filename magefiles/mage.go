package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

// pinned tool versions — keep in sync with the CI workflow and the plan's Global Constraints.
const (
	golangciLintVersion = "v2.12.2"
	mockeryVersion      = "v2.53.4"
)

// binDir is the repo-local tool directory (./bin), matching `make install-deps`.
func binDir() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "bin")
}

// goInstall runs `go install` with GOBIN pointed at ./bin.
func goInstall(pkg string) error {
	return sh.RunWith(map[string]string{"GOBIN": binDir()}, "go", "install", pkg)
}

// Tools installs golangci-lint and mockery into ./bin at pinned versions.
func Tools() error {
	if err := goInstall("github.com/golangci/golangci-lint/v2/cmd/golangci-lint@" + golangciLintVersion); err != nil {
		return fmt.Errorf("install golangci-lint: %w", err)
	}
	if err := goInstall("github.com/vektra/mockery/v2@" + mockeryVersion); err != nil {
		return fmt.Errorf("install mockery: %w", err)
	}
	return nil
}

// Lint runs golangci-lint over the whole module.
func Lint() error {
	return sh.RunV(filepath.Join(binDir(), "golangci-lint"), "run", "./...")
}

// Test runs the full test suite with the race detector.
func Test() error {
	return sh.RunV("go", "test", "-race", "./...")
}

// Mocks regenerates all mockery mocks from .mockery.yaml.
func Mocks() error {
	return sh.RunV(filepath.Join(binDir(), "mockery"))
}

// MocksCheck regenerates mocks and fails if the working tree changed (CI drift guard).
func MocksCheck() error {
	if err := Mocks(); err != nil {
		return err
	}
	return sh.RunV("git", "diff", "--exit-code", "--", "**/mocks/**")
}

// CI runs the full check suite: lint, mock-drift, then tests. Mock-drift goes
// before the (much slower) tests so stale mocks fail fast, and the tests then
// run against freshly regenerated mocks.
func CI() error {
	if err := Lint(); err != nil {
		return err
	}
	if err := MocksCheck(); err != nil {
		return err
	}
	return Test()
}
