# s3gw-testing

![License](https://img.shields.io/github/license/giubacc/s3gw-system-tests)
![Lint](https://github.com/giubacc/s3gw-system-tests/actions/workflows/lint.yaml/badge.svg)
![Nightly Tests](https://github.com/giubacc/s3gw-system-tests/actions/workflows/nightly-tests.yaml/badge.svg)

## Local setup

### Bootstrap

> **Before doing anything else**: ensure to execute the following command
> after the clone:

```shell
git submodule update --init --recursive
```

### Requirements

- Docker, Docker compose
- Helm
- k3d
- kubectl
- Go (1.20+)
- Ginkgo

### Build the s3gw's images

Before creating the `k3d-s3gw-acceptance` cluster,
you have to build the images:

```shell
make build-images
```

> **Be patient**: this will take long.

After the command completes successfully,
you will see the following images:

```shell
docker images
```

- `quay.io/s3gw/s3gw:{@TAG}`
- `quay.io/s3gw/s3gw-ui:{@TAG}`
- `quay.io/s3gw/s3gw-cosi-driver:{@TAG}`
- `quay.io/s3gw/s3gw-cosi-sidecar:{@TAG}`

Where `{@TAG}` is the evaluation of the following expression:

```bash
$(git describe --tags --always)
```

### Create the acceptance cluster

You create the `k3d-s3gw-acceptance` cluster with:

```shell
make acceptance-cluster-create
```

> **WARNING**: the command updates your `.kube/config` with the credentials of
> the just created `k3d-s3gw-acceptance` cluster and sets its context as default.

### Delete the acceptance cluster

```shell
make acceptance-cluster-delete
```

### Prepare the acceptance cluster

You prepare the acceptance cluster with:

```shell
make acceptance-cluster-prepare
```

This imports the pre-built s3gw's images into k3d and triggers
a deployment of needed resources.

### Deploy the s3gw-acceptance-0/s3gw-0 instance on the acceptance cluster

You deploy the `s3gw-acceptance-0/s3gw-0` instance in the acceptance
cluster with:

```shell
make acceptance-cluster-s3gw-deploy
```

Acceptance tests are **NOT** relying on the `s3gw-acceptance-0/s3gw-0` instance.

### Trigger tests on the acceptance cluster

```shell
make acceptance-test-install
```

## Acceptance tests

### Installation & Upgrade tests

When releasing a new s3gw version it is appropriate to run a series of
specialized tests dedicated to ensure the installation and
the upgrade correctness of s3gw the in the Kubernetes cluster.

#### Tag based release tests

With a specific Tag pattern, you can trigger a specific
test workflow involving:

- a **PREVIOUS** `helm charts` version
- a **TARGET** `helm charts` version
- a **PREVIOUS** `image` version
- a **TARGET** `image` version

You can also specify if all the s3gw's images should be built from the
local checkout in the respective submodule.

#### Tag pattern

The pattern recognized by the `Tag based release tests` workflow is the
following:

```text
CP.CP.CP_C.C.C_IP.IP.IP_I.I.I_[IMPORT|NIMPORT]
```

where:

- `CP` is the **PREVIOUS** `helm charts` version
- `C` is the **TARGET** `helm charts` version
- `IP` is the **PREVIOUS** `image` version
- `I` is the **TARGET** `image` version
- `IMPORT` directive mandates to **build & import** the `TARGET` images (`I`)
- `NIMPORT` directive mandates to **pull** the `TARGET` images (`I`) from the registry.

#### Examples

Suppose you want to trigger the `Tag based release tests` workflow with
the purpose of acceptance-testing the not yet released version `0.18.0` of s3gw.
You want, specifically, tests to be executed taking in consideration that
the upgrade from the latest released version, eg: `0.17.0` is correctly handled.
In this case you build the Tag with the following values:

- `CP` : `0.17.0`
- `C` : `0.18.0`
- `IP`: `0.17.0`
- `I` : `0.18.0`
- `IMPORT`

You specify `IMPORT` in the hypothesis that the `0.18.0` s3gw's images still
don't exist on the registry. In this case, the directive triggers a local
build based on the current submodules revision.

The final Tag assumes the following value:

```text
0.17.0_0.18.0_0.17.0_0.18.0_IMPORT
```

So, operatively, what you have to do is:

```shell
git checkout my-testing-branch
git tag 0.17.0_0.18.0_0.17.0_0.18.0_IMPORT
git push origin 0.17.0_0.18.0_0.17.0_0.18.0_IMPORT
```

Once the Tag has been pushed on github, you can follow the execution of the
testing workflow.

## LicenseS

Copyright (c) 2022-2023 [SUSE, LLC](http://suse.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
