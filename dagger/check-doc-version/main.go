// The CheckDocVersion module is designed to check if the version of the
// documentation exists for the version specified in the release-please manifest.
// This is used to ensure that we do not release a new version of the plugin
// without having the corresponding documentation ready.

package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/check-doc-version/internal/dagger"
)

type CheckDocVersion struct{}

// HasVersionDocumentation checks if a version of the documentation exists for the
// version in the release-please manifest.
func (m *CheckDocVersion) HasVersionDocumentation(ctx context.Context, src *dagger.Directory) (bool, error) {
	releasePleaseManifest := ".release-please-manifest.json"
	docusaurusVersions := "web/versions.json"
	ctr := dag.Container().From("alpine:latest").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"apk", "add", "jq"})
	nextVersion, err := ctr.
		WithExec([]string{"jq", "-r", ".[\".\"]", releasePleaseManifest}).
		Stdout(ctx)
	nextVersion = strings.TrimSpace(nextVersion)
	if err != nil {
		return false, fmt.Errorf("cannot find proposed release-please version in %v: %w", releasePleaseManifest,
			err)
	}
	currVersion, err := ctr.WithExec([]string{"jq", "-r", fmt.Sprintf(". | index(\"%v\")", nextVersion),
		docusaurusVersions}).Stdout(ctx)
	currVersion = strings.TrimSpace(currVersion)
	if err != nil {
		return false, fmt.Errorf("error querying versions in %v: %w", docusaurusVersions, err)
	}
	if currVersion == "null" {
		return false, nil
	}
	return true, nil
}
