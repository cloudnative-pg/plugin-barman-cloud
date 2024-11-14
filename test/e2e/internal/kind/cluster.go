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
	"strings"
)

// IsClusterRunning checks if a Kind cluster with the given name is running
func IsClusterRunning(clusterName string) (bool, error) {
	cmd := exec.Command(Kind, "get", "clusters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to get Kind clusters: %w, output: %s", err, string(output))
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if cluster == clusterName {
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
func CreateCluster(name string, opts ...CreateClusterOption) error {
	options := &CreateClusterOptions{}
	for _, opt := range opts {
		opt(options)
	}

	args := []string{"create", "cluster", "--name", name}
	if options.ConfigFile != "" {
		args = append(args, "--config", options.ConfigFile)
	}
	if options.K8sVersion != "" {
		args = append(args, "--image", fmt.Sprintf("kindest/node:%s", options.K8sVersion))
	}

	cmd := exec.Command(Kind, args...) // #nosec
	cmd.Dir, _ = os.Getwd()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("'kind create cluster' failed: %w, output: %s", err, string(output))
	}

	// Since a cluster can mount additional certificates, we need to make sure they are
	// usable by the nodes in the cluster.
	nodes, err := getNodes(name)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		cmd = exec.Command("docker", "exec", node, "update-ca-certificates") // #nosec
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update CA certificates in node %s: %w, output: %s", node, err, string(output))
		}
	}

	for _, network := range options.Networks {
		for _, node := range nodes {
			cmd = exec.Command("docker", "network", "connect", network, node) // #nosec
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to connect node %s to network %s: %w, output: %s", node, network, err,
					string(output))
			}
		}
	}

	return nil
}

func getNodes(clusterName string) ([]string, error) {
	cmd := exec.Command(Kind, "get", "nodes", "--name", clusterName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kind nodes: %w, output: %s", err, string(output))
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}
