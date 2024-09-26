// A generated module for Gotest functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"fmt"

	"dagger/gotest/internal/dagger"
)

type Gotest struct {
	// +private
	Ctr *dagger.Container
	// +private
	KubeVersion string
}

func New(
	// Go version
	//
	// +optional
	// +default="latest"
	goVersion string,
	// setup-envtest version
	// +optional
	// +default="0.19.0"
	setupEnvtestVersion string,
	// Kubernetes version
	// +optional
	// +default="1.31.0"
	kubeVersion string,
	// Container to run the tests
	// +optional
	ctr *dagger.Container,
) *Gotest {
	if ctr != nil {
		return &Gotest{Ctr: ctr}
	}

	user := "noroot"
	modCachePath := fmt.Sprintf("/home/%s/go/pkg/mod", user)
	goCachePath := fmt.Sprintf("/home/%s/.cache/go-build", user)
	ctr = dag.Container().From("golang:"+goVersion).
		WithExec([]string{"curl", "-L",
			fmt.Sprintf("https://dl.k8s.io/release/v%v/bin/linux/amd64/kubectl", kubeVersion),
			"-o", "/usr/local/bin/kubectl"}).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/kubectl"}).
		WithExec([]string{"curl", "-L",
			fmt.Sprintf(
				"https://github.com/kubernetes-sigs/controller-runtime/releases/download/v%v/setup-envtest-linux-amd64",
				setupEnvtestVersion),
			"-o", "/usr/local/bin/setup-envtest"}).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/setup-envtest"}).
		WithExec([]string{"useradd", "-m", user}).
		WithUser(user).
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOMODCACHE", modCachePath).
		WithEnvVariable("GOCACHE", goCachePath).
		WithMountedCache(modCachePath, dag.CacheVolume("go-mod"),
			dagger.ContainerWithMountedCacheOpts{Owner: user}).
		WithMountedCache(goCachePath, dag.CacheVolume("go-build"),
			dagger.ContainerWithMountedCacheOpts{Owner: user})

	return &Gotest{Ctr: ctr, KubeVersion: kubeVersion}
}

func (m *Gotest) UnitTest(
	ctx context.Context,
	// Source directory
	// +required
	src *dagger.Directory,
) (string, error) {
	envtestCmd := []string{"setup-envtest", "use", "-p", "path", m.KubeVersion}
	return m.Ctr.WithDirectory("/src", src).
		// Setup envtest. There is no proper way to install it from a git release, so we use the go install command
		WithExec(envtestCmd).
		WithEnvVariable("KUBEBUILDER_ASSETS",
			fmt.Sprintf("/home/noroot/.local/share/kubebuilder-envtest/k8s/%v-linux-amd64", m.KubeVersion),
		).
		WithWorkdir("/src").
		// Exclude the e2e tests, we don't want to run them here
		WithoutDirectory("/src/test/e2e").
		WithExec([]string{"go", "test", "./..."}).Stdout(ctx)
}
