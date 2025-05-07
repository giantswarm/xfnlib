package aws

import (
	"context"
	"os"

	"github.com/crossplane-contrib/provider-aws/pkg/utils/pointer"
	"github.com/go-ini/ini"

	"github.com/crossplane/function-sdk-go/logging"
	"github.com/giantswarm/xfnlib/pkg/auth/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	credsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	stscredsv2 "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/giantswarm/xfnlib/pkg/composite"
)

type endpoint struct {
	Services          []string `json:"services,omitempty"`
	HostnameImmutable bool     `json:"hostnameImmutable,omitempty"`

	URL *struct {
		Type    string `json:"type,omitempty"`
		Dynamic string `json:"dynamic,omitempty"`
		Static  string `json:"static,omitempty"`
	} `json:"url,omitempty"`
}

type credentials struct {
	Source          string `json:"source,omitempty"`
	AccessKeyID     string `json:"accessKeyID,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	SessionToken    string `json:"sessionToken,omitempty"`
	SecretRef       struct {
		Name      string `json:"name,omitempty"`
		Namespace string `json:"namespace,omitempty"`
		Key       string `json:"key,omitempty"`
	} `json:"secretRef,omitempty"`
	Upbound struct{} `json:"upbound,omitempty"`

	WebIdentity struct {
		RoleArn string `json:"roleArn,omitempty"`
	} `json:"webIdentity,omitempty"`
}

type ProviderConfigSpec struct {
	Endpoint        *endpoint    `json:"endpoint"`
	Credentials     *credentials `json:"credentials"`
	AssumeRoleChain []struct {
		RoleARN string `json:"roleARN"`
	} `json:"assumeRoleChain,omitempty"`

	S3UsePathStyle            bool `json:"s3_use_path_style,omitempty"`
	SkipCredentialsValidation bool `json:"skip_credentials_validation,omitempty"`
	SkipRegionValidation      bool `json:"skip_region_validation,omitempty"`
	SkipRequestingAccountID   bool `json:"skip_requesting_account_id,omitempty"`
	SkipMetadataAPICheck      bool `json:"skip_metadata_api_check,omitempty"`
}

// GetAssumeRoleArn retrieves the current provider role arn from providerconfig
//
// This requires the service account the function is running with to have
// additional permissions in order to obtain the `providerconfig`
//
// In order to retrieve the providerconfig, the service account running this
// function must be bound to a role allowing:
//
//		rules:
//		- apiGroups:
//		  - aws.upbound.io
//		  resources:
//		  - providerconfigs
//		  verbs:
//		  - get
//	 - apiGroups:
//		  - ""
//		  resources:
//		  - secrets
//		  verbs:
//		  - get
func GetProviderConfig(providerConfigRef *string) (cfg *ProviderConfigSpec, err error) {
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

	var spec ProviderConfigSpec
	if err = composite.To(u.Object["spec"], &spec); err != nil {
		err = errors.Wrapf(err, "unable to decode provider config")
		return
	}

	cfg = &spec
	return
}

func GetCredentialsFromSecret(name, namespace, key string) (creds credsv2.StaticCredentialsProvider, err error) {
	var (
		cl     client.Client
		ok     bool
		secret = corev1.Secret{}
		data   []byte
	)

	if cl, err = kubernetes.Client(); err != nil {
		err = errors.Wrap(err, "error setting up kubernetes client")
		return
	}

	if err = cl.Get(context.Background(), client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &secret); err != nil {
		err = errors.Wrapf(err, "failed to load secret %s in namespace %s", name, namespace)
		return
	}

	if data, ok = secret.Data[key]; !ok {
		err = errors.Wrapf(err, "failed to load key %s in secret %s in namespace %s", key, name, namespace)
		return
	}

	var iniFile *ini.File
	{
		if iniFile, err = ini.Load(data); err != nil {
			return
		}
	}

	var section *ini.Section
	{
		if section, err = iniFile.GetSection("default"); err != nil {
			return
		}
	}

	var accesskey, secretkey, session string
	{
		var a, s, se *ini.Key
		if section.HasKey("aws_access_key_id") {
			if a, err = section.GetKey("aws_access_key_id"); err != nil {
				return
			}
			accesskey = a.String()
		}

		if section.HasKey("aws_secret_access_key") {
			if s, err = section.GetKey("aws_secret_access_key"); err != nil {
				return
			}
			secretkey = s.String()
		}

		if section.HasKey("aws_session_token") {
			if se, err = section.GetKey("aws_session_token"); err != nil {
				return
			}
			session = se.String()
		}
	}

	creds = credsv2.NewStaticCredentialsProvider(accesskey, secretkey, session)
	return
}

// Config sets up the AWS config using assume roles
//
// For this method to work, the service account the function is running with
// must be annotated with
//
//	annotations:
//	  eks.amazonaws.com/role-arn: YOUR_ROLE_ARN
func Config(region, providerConfigRef *string, log logging.Logger) (cfg aws.Config, services map[string]string, err error) {
	var (
		ctx           context.Context = context.TODO()
		pcfg          *ProviderConfigSpec
		assumeRoleArn *string
		opts          []config.LoadOptionsFunc = make([]config.LoadOptionsFunc, 0)
	)

	services = make(map[string]string)

	if region != nil {
		opts = append(opts, config.WithRegion(*region))
	}

	if pcfg, err = GetProviderConfig(providerConfigRef); err != nil {
		err = errors.Wrap(err, "unable to get assumerole")
		return
	}

	if pcfg.Endpoint != nil {
		log.Info("setting up endpoint")
		var epopts []config.LoadOptionsFunc
		epopts, err = getEndpointOptions(pcfg.Endpoint)
		if err != nil {
			err = errors.Wrap(err, "unable to get endpoint options")
			return
		}
		opts = append(opts, epopts...)

		if pcfg.Endpoint.Services != nil {
			for _, service := range pcfg.Endpoint.Services {
				services[service] = pcfg.Endpoint.URL.Static
			}
		}
		log.Info("ProviderConfig", "endpoint", pcfg.Endpoint)
	}

	if pcfg.Credentials.Source == "Secret" {
		var creds credsv2.StaticCredentialsProvider
		creds, err = GetCredentialsFromSecret(
			pcfg.Credentials.SecretRef.Name,
			pcfg.Credentials.SecretRef.Namespace,
			pcfg.Credentials.SecretRef.Key,
		)
		if err != nil {
			err = errors.Wrap(err, "unable to get credentials from secret")
			return
		}

		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	if cfg, err = config.LoadDefaultConfig(
		ctx, func(cfg *config.LoadOptions) error {
			for _, opt := range opts {
				if err := opt(cfg); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
		err = errors.Wrapf(err, "failed to load initial aws config for region %q", *region)
		return
	}

	switch pcfg.Credentials.Source {
	case "Secret":
		return
	case "Upbound":
		err = errors.New("upbound credentials not supported")
		return
	case "WebIdentity":
		log.Info("Using WebIdentity credentials")
		stsclient := sts.NewFromConfig(cfg)

		if len(pcfg.AssumeRoleChain) > 0 {
			assumeRoleArn = &pcfg.AssumeRoleChain[0].RoleARN

			log.Info("Assuming role", "role", *assumeRoleArn)
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
		} else {
			awscfg, err := config.LoadDefaultConfig(ctx, nil)
			if err != nil {
				err = errors.Wrap(err, "failed to load default AWS config")
			}

			roleArn := pcfg.Credentials.WebIdentity.RoleArn

			stsclient := sts.NewFromConfig(awscfg)

			cfg, err = config.LoadDefaultConfig(
				ctx,
				config.WithRegion(*region),
				config.WithCredentialsProvider(aws.NewCredentialsCache(
					stscreds.NewWebIdentityRoleProvider(
						stsclient,
						pointer.StringValue(&roleArn),
						stscreds.IdentityTokenFile(getWebidentityTokenFilePath()),
						func(o *stscreds.WebIdentityRoleOptions) {
							o.RoleSessionName = "crossplane-provider-aws"
						},
					)),
				),
			)
		}
	}

	return
}

const webIdentityTokenFileDefaultPath = "/var/run/secrets/eks.amazonaws.com/serviceaccount/token"

func getWebidentityTokenFilePath() string {
	if path := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"); path != "" {
		return path
	}
	return webIdentityTokenFileDefaultPath
}

func getEndpointOptions(ep *endpoint) (opts []config.LoadOptionsFunc, err error) {
	if ep.URL != nil {
		switch ep.URL.Type {
		case "dynamic":
			if ep.URL.Dynamic != "" {
				opts = append(opts, config.WithEndpointResolverWithOptions(
					aws.EndpointResolverWithOptionsFunc(
						func(service, region string, options ...interface{}) (aws.Endpoint, error) {
							return aws.Endpoint{
								URL:               ep.URL.Dynamic,
								HostnameImmutable: ep.HostnameImmutable,
							}, nil
						},
					),
				))
			}
		case "static":
			if ep.URL.Static != "" {
				opts = append(opts, config.WithEndpointResolverWithOptions(
					aws.EndpointResolverWithOptionsFunc(
						func(service, region string, options ...interface{}) (aws.Endpoint, error) {
							return aws.Endpoint{
								URL:               ep.URL.Static,
								HostnameImmutable: ep.HostnameImmutable,
							}, nil
						},
					),
				))
			}
		}
	}
	return
}
