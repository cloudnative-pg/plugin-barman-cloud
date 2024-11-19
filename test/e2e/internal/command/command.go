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

package command

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// TODO: extract this and the one in CloudNativePG to a common library

// ContainerLocator is a struct that contains the information needed to locate a container in a pod.
type ContainerLocator struct {
	NamespaceName string
	PodName       string
	ContainerName string
}

// ExecuteInContainer executes a command in a container. If timeout is not nil, the command will be
// executed with the specified timeout. The function returns the stdout and stderr of the command.
func ExecuteInContainer(
	ctx context.Context,
	clientSet kubernetes.Clientset,
	cfg *rest.Config,
	container ContainerLocator,
	timeout *time.Duration,
	command []string,
) (string, string, error) {
	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(container.PodName).
		Namespace(container.NamespaceName).
		SubResource("exec").
		Param("container", container.ContainerName).
		Param("stdout", "true").
		Param("stderr", "true")
	for _, cmd := range command {
		req.Param("command", cmd)
	}

	newConfig := *cfg // local copy avoids modifying the passed config arg
	if timeout != nil {
		req.Timeout(*timeout)
		newConfig.Timeout = *timeout
		timedCtx, cancelFunc := context.WithTimeout(ctx, *timeout)
		defer cancelFunc()
		ctx = timedCtx
	}

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("error creating executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", fmt.Errorf("error executing command in pod '%s/%s': %w",
			container.NamespaceName, container.PodName, err)
	}

	return stdout.String(), stderr.String(), nil
}
