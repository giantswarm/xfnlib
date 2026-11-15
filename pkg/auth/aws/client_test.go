package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
)

func TestWebIdentityConfigStopsWhenDefaultConfigCannotLoad(t *testing.T) {
	// A profile that exists in no shared config file makes
	// config.LoadDefaultConfig fail before any role is assumed.
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	t.Setenv("AWS_PROFILE", "xfnlib-test-profile-that-does-not-exist")

	_, err := webIdentityConfig(context.Background(), "eu-west-1", "arn:aws:iam::123456789012:role/example")
	if err == nil {
		t.Fatal("expected an error when the default AWS config cannot be loaded, got nil")
	}

	if want := "failed to load default AWS config"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected the error to report %q, got %q", want, err)
	}
}

func TestGetEndpointOptionsSetsBaseEndpoint(t *testing.T) {
	tests := map[string]struct {
		urlType, static, dynamic string
		want                     string
	}{
		"static url":          {urlType: "static", static: "http://localhost.localstack.cloud:4566", want: "http://localhost.localstack.cloud:4566"},
		"dynamic url":         {urlType: "dynamic", dynamic: "https://example.internal", want: "https://example.internal"},
		"static without url":  {urlType: "static"},
		"dynamic without url": {urlType: "dynamic"},
		"unknown type":        {urlType: "other", static: "http://ignored", dynamic: "http://ignored"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ep := &endpoint{}
			ep.URL = &struct {
				Type    string `json:"type,omitempty"`
				Dynamic string `json:"dynamic,omitempty"`
				Static  string `json:"static,omitempty"`
			}{Type: tc.urlType, Static: tc.static, Dynamic: tc.dynamic}

			opts, err := getEndpointOptions(ep)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var lo config.LoadOptions
			for _, opt := range opts {
				if err := opt(&lo); err != nil {
					t.Fatalf("applying option: %v", err)
				}
			}

			if lo.BaseEndpoint != tc.want {
				t.Fatalf("expected base endpoint %q, got %q", tc.want, lo.BaseEndpoint)
			}
		})
	}
}

func TestGetEndpointOptionsWithoutURL(t *testing.T) {
	opts, err := getEndpointOptions(&endpoint{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("expected no options without a URL, got %d", len(opts))
	}
}
