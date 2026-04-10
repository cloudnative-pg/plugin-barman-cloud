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
	"strings"
	"testing"
)

func TestParseTimelineIDFromPgControldataOutput(t *testing.T) {
	pgData := "/var/lib/postgresql/data/pgdata"

	tests := []struct {
		name         string
		out          string
		want         int
		wantErr      bool
		errHasPGData bool // if true, error must mention pgData path (parse-not-found cases)
	}{
		{
			name: "typical_pg_controldata_snippet",
			out: `
Database cluster state:               in production
Latest checkpoint location:           0/3000028
Latest checkpoint's REDO location:    0/3000028
Latest checkpoint's TimeLineID:       2
Latest checkpoint's REDO WAL file:    000000010000000000000003
`,
			want:    2,
			wantErr: false,
		},
		{
			name: "timeline_one",
			out: `Latest checkpoint's TimeLineID:       1
`,
			want:    1,
			wantErr: false,
		},
		{
			name:         "missing_timeline_line",
			out:          "Database cluster state: in production\n",
			want:         0,
			wantErr:      true,
			errHasPGData: true,
		},
		{
			name:         "empty",
			out:          "",
			want:         0,
			wantErr:      true,
			errHasPGData: true,
		},
		{
			name: "overflow_timeline",
			out: `Latest checkpoint's TimeLineID:       999999999999999999999999999999999999
`,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimelineIDFromPgControldataOutput(tt.out, pgData)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errHasPGData && !strings.Contains(err.Error(), pgData) {
					t.Errorf("error should mention PGDATA path: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
