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

// Package scheme provides utilities for building runtime schemes
// with support for custom CNPG API groups.
package scheme

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	crscheme "sigs.k8s.io/controller-runtime/pkg/scheme"
)

// AddCNPGToScheme registers CNPG types into the given scheme using
// the API group configured via CUSTOM_CNPG_GROUP/CUSTOM_CNPG_VERSION
// environment variables, defaulting to postgresql.cnpg.io/v1.
// This allows the plugin to work with any CNPG-based operator.
func AddCNPGToScheme(ctx context.Context, s *runtime.Scheme) {
	cnpgGroup := viper.GetString("custom-cnpg-group")
	cnpgVersion := viper.GetString("custom-cnpg-version")
	if len(cnpgGroup) == 0 {
		cnpgGroup = cnpgv1.SchemeGroupVersion.Group
	}
	if len(cnpgVersion) == 0 {
		cnpgVersion = cnpgv1.SchemeGroupVersion.Version
	}

	schemeGroupVersion := schema.GroupVersion{Group: cnpgGroup, Version: cnpgVersion}
	schemeBuilder := &crscheme.Builder{GroupVersion: schemeGroupVersion}
	schemeBuilder.Register(&cnpgv1.Cluster{}, &cnpgv1.ClusterList{})
	schemeBuilder.Register(&cnpgv1.Backup{}, &cnpgv1.BackupList{})
	schemeBuilder.Register(&cnpgv1.ScheduledBackup{}, &cnpgv1.ScheduledBackupList{})
	utilruntime.Must(schemeBuilder.AddToScheme(s))

	log.FromContext(ctx).Info("CNPG types registration", "schemeGroupVersion", schemeGroupVersion)
}
