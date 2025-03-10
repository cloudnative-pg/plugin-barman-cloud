package instance

import (
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

type backupResultMetadata struct {
	timeline    string
	version     string
	name        string
	displayName string
	clusterUID  string
	pluginName  string
}

func (b backupResultMetadata) toMap() map[string]string {
	return map[string]string{
		"timeline":    b.timeline,
		"version":     b.version,
		"name":        b.name,
		"displayName": b.displayName,
		"clusterUID":  b.clusterUID,
		"pluginName":  b.pluginName,
	}
}

func newBackupResultMetadata(clusterUID types.UID, timeline int) backupResultMetadata {
	return backupResultMetadata{
		timeline:   strconv.Itoa(timeline),
		clusterUID: string(clusterUID),
		// static values
		version:     metadata.Data.Version,
		name:        metadata.Data.Name,
		displayName: metadata.Data.DisplayName,
		pluginName:  metadata.PluginName,
	}
}

func newBackupResultMetadataFromMap(m map[string]string) backupResultMetadata {
	if m == nil {
		return backupResultMetadata{}
	}

	return backupResultMetadata{
		timeline:    m["timeline"],
		version:     m["version"],
		name:        m["name"],
		displayName: m["displayName"],
		clusterUID:  m["clusterUID"],
		pluginName:  m["pluginName"],
	}
}
