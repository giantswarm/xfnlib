# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

### Changed

- Update dependencies
- Allow assuming a role directly on AWS, no need for intermediate role.
- Set the provider config's endpoint override as the AWS base endpoint instead
  of the deprecated global endpoint resolver. `hostnameImmutable` is no longer
  applied at config level: S3 clients built from the returned config must
  honour the provider config's `s3_use_path_style` to keep the bucket out of
  the hostname.

### Fixed

- Return an error from web identity credential setup when the default AWS
  config cannot be loaded, instead of carrying on with an empty config.

[Unreleased]: https://github.com/giantswarm/xfnlib/tree/main
- Regenerate `.github/workflows/zz_generated.*.yaml` via devctl to use the centralized reusable workflow, removing the Node-20 `mindsers/changelog-reader-action` dependency.
