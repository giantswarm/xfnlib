package aws

import (
	"context"

	"github.com/giantswarm/xfnlib/pkg/auth/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	stscredsv2 "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/giantswarm/xfnlib/pkg/composite"
)

// GetAssumeRoleArn retrieves the current provider role arn from the providerconfig
//
// This requires the service account the function is runmning
func GetAssumeRoleArn(providerConfigRef *string) (arn *string, err error) {
	var (
		unstructuredData *unstructured.Unstructured = &unstructured.Unstructured{}
		cl               client.Client
	)
	if cl, err = kubernetes.Client(); err != nil {
		err = errors.Wrap(err, "error setting up kubernetes client")
		return
	}
	// Get the provider context
	unstructuredData.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "aws.upbound.io",
		Kind:    "ProviderConfig",
		Version: "v1beta1",
	})

	if err = cl.Get(context.Background(), client.ObjectKey{
		Name: *providerConfigRef,
	}, unstructuredData); err != nil {
		err = errors.Wrap(err, "failed to load providerconfig")
		return
	}

	type _spec struct {
		AssumeRoleChain []struct {
			RoleARN string `json:"roleARN"`
		} `json:"assumeRoleChain"`
	}

	var spec _spec
	if err = composite.To(unstructuredData.Object["spec"], &spec); err != nil {
		err = errors.Wrapf(err, "unable to decode provider config")
		return
	}

	// We only care about the first in the chain here.
	arn = &spec.AssumeRoleChain[0].RoleARN
	return
}

func Config(region, providerConfigRef *string) (cfg aws.Config, err error) {
	var (
		ctx           context.Context = context.TODO()
		acfg          aws.Config
		assumeRoleArn *string
	)

	if assumeRoleArn, err = GetAssumeRoleArn(providerConfigRef); err != nil {
		return
	}

	// Set up the assume role clients
	if acfg, err = config.LoadDefaultConfig(
		ctx, config.WithRegion(*region),
	); err != nil {
		err = errors.Wrap(err, "failed to load initial aws config")
		return
	}
	stsclient := sts.NewFromConfig(acfg)

	if cfg, err = config.LoadDefaultConfig(
		ctx,
		config.WithRegion(*region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			stscredsv2.NewAssumeRoleProvider(
				stsclient,
				*assumeRoleArn,
			)),
		),
	); err != nil {
		err = errors.Wrap(err, "failed to load aws config for assume role")
	}
	return
}
