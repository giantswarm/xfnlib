package kubernetes

import (
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Get a kubernetes client for cluster operations
//
// By default funtions have no permissions to the cluster and must be explicitly
// granted any additional required permissions.
//
// Where a function requires access to cluster resources, the set should be kept
// to the smallest feasible set to ensure that no errant function is able to
// access information inside the cluster that it shouldn't be able to.
func Client() (c client.Client, err error) {
	var config *rest.Config

	if config, err = clientconfig.GetConfig(); err != nil {
		err = errors.Wrap(err, "cannot get cluster config")
		return
	}

	if c, err = client.New(config, client.Options{}); err != nil {
		err = errors.Wrap(err, "failed to create cluster client")
	}

	return
}
