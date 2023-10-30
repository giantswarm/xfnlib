package kubernetes

import (
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

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
