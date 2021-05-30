# E2E Test

This document is to help developers understand how for run e2e test CAPDO.

## Requirements

In order to run the e2e tests the following requirements must be met:

* Ginkgo
* Docker
* Kind v0.7.0+

### Environment variables

The first step to running the e2e tests is setting up the required environment variables:

| Environment variable              | Description                                                                                           |
| --------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `LINODE_CLI_TOKEN`       | The Linode API V2 access token                                                                  |
| `LINODE_CONTROL_PLANE_MACHINE_IMAGE`  | The Linode Image id or slug                                                                     |
| `LINODE_NODE_MACHINE_IMAGE`           | The Linode Image id or slug                                                                     |
| `LINODE_SSH_KEY_FINGERPRINT`          | The ssh key id or fingerprint (Should be already registered in the Linode Account)

### Running e2e test

In the root project directory run:

```
make test-e2e
```

### Running Conformance test

In the root project directory run:

```
make test-conformance
```