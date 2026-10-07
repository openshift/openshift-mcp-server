package common

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfigPath creates rest.Config from the provided path for kubeconfig.
// If the provided path is empty, the in-cluster config is the default.
func GetKubeConfig(kubeConfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	if kubeConfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig file %q: %w", kubeConfigPath, err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	}
	return config, nil
}
