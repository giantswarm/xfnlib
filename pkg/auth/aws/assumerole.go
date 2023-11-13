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

// GetAssumeRoleArn retrieves the current provider role arn from providerconfig
//
// This requires the service account the function is running with to have
// additional permissions in order to obtain the `providerconfig`
//
// In order to retrieve the providerconfig, the service account running this
// function must be bound to a role allowing:
//
//	rules:
//	- apiGroups:
//	  - aws.upbound.io
//	  resources:
//	  - providerconfigs
//	  verbs:
//	  - get
func GetAssumeRoleArn(providerConfigRef *string) (arn *string, err error) {
	var (
		u  *unstructured.Unstructured = &unstructured.Unstructured{}
		cl client.Client
	)
	if cl, err = kubernetes.Client(); err != nil {
		err = errors.Wrap(err, "error setting up kubernetes client")
		return
	}
	// Get the provider context
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "aws.upbound.io",
		Kind:    "ProviderConfig",
		Version: "v1beta1",
	})

	if err = cl.Get(context.Background(), client.ObjectKey{
		Name: *providerConfigRef,
	}, u); err != nil {
		err = errors.Wrapf(err, "failed to load providerconfig %s", *providerConfigRef)
		return
	}

	type _spec struct {
		AssumeRoleChain []struct {
			RoleARN string `json:"roleARN"`
		} `json:"assumeRoleChain"`
	}

	var spec _spec
	if err = composite.To(u.Object["spec"], &spec); err != nil {
		err = errors.Wrapf(err, "unable to decode provider config")
		return
	}

	// We only care about the first in the chain here.
	arn = &spec.AssumeRoleChain[0].RoleARN
	return
}

// Config sets up the AWS config using assume roles
//
// For this method to work, the service account the function is running with
// must be annotated with
//
//	annotations:
//	  eks.amazonaws.com/role-arn: YOUR_ROLE_ARN
func Config(region, providerConfigRef *string) (cfg aws.Config, err error) {
	var (
		ctx           context.Context = context.TODO()
		acfg          aws.Config
		assumeRoleArn *string
	)

	if assumeRoleArn, err = GetAssumeRoleArn(providerConfigRef); err != nil {
		err = errors.Wrap(err, "unable to get assumerole")
		return
	}

	// Set up the assume role clients
	if acfg, err = config.LoadDefaultConfig(
		ctx, config.WithRegion(*region),
	); err != nil {
		err = errors.Wrapf(err, "failed to load initial aws config for region %q", *region)
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
		err = errors.Wrapf(err, "failed to load aws config for assume role '%q'", *assumeRoleArn)
	}
	return
}
