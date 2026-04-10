/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudnative-pg/machinery/pkg/log"
)

var timelineRe = regexp.MustCompile(`Latest checkpoint's TimeLineID:\s+(\d+)`)

// currentTimeline returns the server's current PostgreSQL timeline by
// parsing pg_controldata output.
//
// This is reliable for the promotion case because PostgreSQL performs a
// synchronous end-of-recovery checkpoint (which updates the control file)
// before the server starts accepting connections and before the WAL
// archiver is signaled. By the time this function is called during the
// first WAL archive attempt, the control file reflects the promoted
// timeline.
//
// Returns an error if the timeline cannot be determined. Callers must NOT
// silently fall back to omitting --timeline, as that reintroduces the
// original "Expected empty archive" bug after failover.
func currentTimeline(ctx context.Context, pgDataPath string) (int, error) {
	contextLogger := log.FromContext(ctx)

	cmd := exec.CommandContext(ctx, "pg_controldata", pgDataPath) // #nosec G204
	cmd.Env = append(cmd.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf(
			"pg_controldata exec failed (PGDATA=%s): %w; "+
				"WAL archive check cannot run safely without a timeline — "+
				"set annotation cnpg.io/skipEmptyWalArchiveCheck=enabled "+
				"as a manual workaround",
			pgDataPath, err)
	}

	tl, err := parseTimelineIDFromPgControldataOutput(string(out), pgDataPath)
	if err != nil {
		return 0, err
	}

	contextLogger.Info("Detected PostgreSQL timeline from pg_controldata",
		"timeline", tl)
	return tl, nil
}

// parseTimelineIDFromPgControldataOutput extracts Latest checkpoint's TimeLineID
// from pg_controldata stdout. pgDataPath is used only in error messages.
func parseTimelineIDFromPgControldataOutput(out string, pgDataPath string) (int, error) {
	matches := timelineRe.FindStringSubmatch(out)
	if len(matches) < 2 {
		return 0, fmt.Errorf(
			"could not parse TimeLineID from pg_controldata output "+
				"(PGDATA=%s); set annotation "+
				"cnpg.io/skipEmptyWalArchiveCheck=enabled as a manual "+
				"workaround",
			pgDataPath)
	}

	tl, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil {
		return 0, fmt.Errorf("parse timeline %q: %w", matches[1], err)
	}

	return tl, nil
}
