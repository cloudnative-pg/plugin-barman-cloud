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
	"context"
	"fmt"
	"os/exec"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

// IsClusterRunning checks if a Kind cluster with the given name is running
func IsClusterRunning(provider *cluster.Provider, clusterName string) (bool, error) {
	clusters, err := provider.List()
	if err != nil {
		return false, fmt.Errorf("failed to list Kind clusters: %w", err)
	}
	for _, c := range clusters {
		if c == clusterName {
			return true, nil
		}
	}

	return false, nil
}

// CreateClusterOptions are the options for creating a Kind cluster
type CreateClusterOptions struct {
	ConfigFile string
	K8sVersion string
	Networks   []string
}

// CreateClusterOption is the option for creating a Kind cluster
type CreateClusterOption func(*CreateClusterOptions)

// WithConfigFile sets the config file for creating a Kind cluster
func WithConfigFile(configFile string) CreateClusterOption {
	return func(opts *CreateClusterOptions) {
		opts.ConfigFile = configFile
	}
}

// WithK8sVersion sets the Kubernetes version for creating a Kind cluster
func WithK8sVersion(k8sVersion string) CreateClusterOption {
	return func(opts *CreateClusterOptions) {
		opts.K8sVersion = k8sVersion
	}
}

// WithNetwork sets the network for creating a Kind cluster
func WithNetworks(networks []string) CreateClusterOption {
	return func(opts *CreateClusterOptions) {
		opts.Networks = networks
	}
}

// CreateCluster creates a Kind cluster with the given name
func CreateCluster(ctx context.Context, provider *cluster.Provider, name string, opts ...CreateClusterOption) error {
	options := &CreateClusterOptions{}
	for _, opt := range opts {
		opt(options)
	}

	createOpts := []cluster.CreateOption{
		cluster.CreateWithRetain(true),
		cluster.CreateWithDisplayUsage(true),
		cluster.CreateWithDisplaySalutation(true),
	}
	if options.ConfigFile != "" {
		createOpts = append(createOpts, cluster.CreateWithConfigFile(options.ConfigFile))
	}
	if options.K8sVersion != "" {
		createOpts = append(createOpts, cluster.CreateWithNodeImage(fmt.Sprintf("kindest/node:%s", options.K8sVersion)))
	}
	err := provider.Create(name, createOpts...)
	if err != nil {
		return fmt.Errorf("kind cluster creation failed: %w", err)
	}

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Since a cluster can mount additional certificates, we need to make sure they are
	// usable by the nodes in the cluster.
	nodeList, err := getNodes(provider, name)
	if err != nil {
		return err
	}
	for _, node := range nodeList {
		cmd := exec.Command("docker", "exec", node.String(), "update-ca-certificates") // #nosec
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update CA certificates in node %s: %w, output: %s", node, err, string(output))
		}

		execConfig := container.ExecOptions{
			Cmd:          strslice.StrSlice([]string{"update-ca-certificates"}),
			AttachStdout: true,
			AttachStderr: true,
		}
		execID, err := cli.ContainerExecCreate(ctx, node.String(), execConfig)
		if err != nil {
			return fmt.Errorf("failed to create exec instance in node %s: %w", node.String(), err)
		}

		err = cli.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{})
		if err != nil {
			return fmt.Errorf("failed to start exec instance in node %s: %w", node.String(), err)
		}
	}

	for _, netw := range options.Networks {
		for _, node := range nodeList {
			err := cli.NetworkConnect(ctx, netw, node.String(), nil)
			if err != nil {
				return fmt.Errorf("failed to connect node %s to network %s: %w", node.String(), netw, err)
			}
		}
	}

	return nil
}

func getNodes(provider *cluster.Provider, clusterName string) ([]nodes.Node, error) {
	nodeList, err := provider.ListNodes(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kind nodes: %w", err)
	}

	return nodeList, nil
}
