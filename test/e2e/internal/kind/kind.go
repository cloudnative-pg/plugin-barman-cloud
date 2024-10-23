/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kind

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Kind is the name of the Kind binary
const Kind = "kind"

// EnsureVersion ensures that there is a Kind binary, is at the correct version, and is in the PATH
func EnsureVersion(requiredVersion string) error {
	if _, err := exec.LookPath(Kind); err == nil {
		version, err := Version()
		if err != nil {
			return err
		}
		if strings.Contains(version, requiredVersion) {
			return nil
		}
	}

	return Install(requiredVersion)
}

// InstallOptions are the options for installing the Kind binary
type InstallOptions struct {
	BinDir string
}

// InstallOption is the option for installing the Kind binary
type InstallOption func(*InstallOptions)

// WithBinDir sets the directory to install the Kind binary
func WithBinDir(binDir string) InstallOption {
	return func(opts *InstallOptions) {
		opts.BinDir = binDir
	}
}

// Install installs the Kind binary in the local project's /bin directory
func Install(version string, opts ...InstallOption) error {
	options := &InstallOptions{
		BinDir: filepath.Join(".", "bin"), // default bin directory
	}
	for _, opt := range opts {
		opt(options)
	}

	// Get BinDir absolute path
	absBinDir, err := filepath.Abs(options.BinDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for bin directory: %w", err)
	}
	options.BinDir = absBinDir

	// Ensure the /bin directory exists
	if err := os.MkdirAll(options.BinDir, os.ModeDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Determine the OS and architecture
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Download the Kind binary
	url := fmt.Sprintf("https://github.com/kubernetes-sigs/kind/releases/download/%s/kind-%s-%s", version, osName, arch)
	binaryPath := filepath.Join(options.BinDir, Kind)
	cmd := exec.Command("curl", "-Lo", binaryPath, url) // #nosec
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download Kind binary: %w, output: %s", err, string(output))
	}

	// Make the Kind binary executable
	cmd = exec.Command("chmod", "+x", binaryPath) // #nosec
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to make Kind binary executable: %w, output: %s", err, string(output))
	}

	if err := os.Setenv("PATH", fmt.Sprintf("%s:%s", options.BinDir, os.Getenv("PATH"))); err != nil {
		return fmt.Errorf("failed to set PATH: %w", err)
	}

	return nil
}

// Version returns the current version of the Kind binary
func Version() (string, error) {
	cmd := exec.Command(Kind, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get Kind version: %w, output: %s", err, string(output))
	}
	version := strings.TrimSpace(string(output))
	return version, nil
}
