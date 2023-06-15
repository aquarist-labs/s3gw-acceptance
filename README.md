# s3gw-acceptance

![License](https://img.shields.io/github/license/aquarist-labs/s3gw-acceptance)
![Lint](https://github.com/aquarist-labs/s3gw-acceptance/actions/workflows/lint.yaml/badge.svg)

<!-- TOC -->

- [s3gw-acceptance](#s3gw-acceptance)
  - [Test Status Overview](#test-status-overview)
    - [Recurring Tests](#recurring-tests)
    - [Upgrade Test Matrix](#upgrade-test-matrix)
  - [Local setup](#local-setup)
    - [Bootstrap](#bootstrap)
    - [Requirements](#requirements)
    - [Create the acceptance cluster](#create-the-acceptance-cluster)
    - [Delete the acceptance cluster](#delete-the-acceptance-cluster)
    - [Build the s3gw's images](#build-the-s3gws-images)
    - [Prepare the acceptance cluster](#prepare-the-acceptance-cluster)
    - [Deploy the s3gw-acceptance-0/s3gw-0 instance on the acceptance cluster](#deploy-the-s3gw-acceptance-0s3gw-0-instance-on-the-acceptance-cluster)
    - [Trigger tests on the acceptance cluster](#trigger-tests-on-the-acceptance-cluster)
  - [Acceptance tests](#acceptance-tests)
    - [Installation \& Upgrade tests](#installation--upgrade-tests)
      - [Tag based triggered tests](#tag-based-triggered-tests)
      - [Tag pattern](#tag-pattern)
      - [Examples](#examples)
  - [License](#license)

<!-- /TOC -->

## Test Status Overview

### Recurring Tests

![Last Release Weekly Tests](https://github.com/aquarist-labs/s3gw-acceptance/actions/workflows/last-release-weekly-tests.yaml/badge.svg)

### Upgrade Test Matrix

| From/To | 0.15.0 | 0.16.0 | 0.17.0 |
|:-------:|:------:|:------:|:------:|
|  0.14.0 |![.](./assets/tests/u0.14.0_0.15.0.svg)|![.](./assets/tests/u0.14.0_0.16.0.svg)|![.](./assets/not-apply.svg)|
|  0.15.0 |        |![.](./assets/tests/u0.15.0_0.16.0.svg)|![.](./assets/not-apply.svg)|
|  0.16.0 |        |        |![.](./assets/not-apply.svg)|

|Sym||
|:--|:--|
|![.](./assets/OK.svg)|passed|
|![.](./assets/KO.svg)|failed|
|![.](./assets/not-apply.svg)|incompatible|

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

### Build the s3gw's images

After you have created the `k3d-s3gw-acceptance` cluster,
you can build the s3gw's images and import those into the cluster:

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

### Prepare the acceptance cluster

You prepare the acceptance cluster with:

```shell
make acceptance-cluster-prepare
```

This triggers the deployment of needed resources.

### Deploy the s3gw-acceptance-0/s3gw-0 instance on the acceptance cluster

Optionally, you can deploy the `s3gw-acceptance-0/s3gw-0` instance in the acceptance
cluster with:

```shell
make acceptance-cluster-s3gw-deploy
```

Note that acceptance tests are **NOT** relying on the `s3gw-acceptance-0/s3gw-0`
instance.

### Trigger tests on the acceptance cluster

```shell
make acceptance-test-install
```

## Acceptance tests

### Installation & Upgrade tests

When releasing a new s3gw version it is appropriate to run a series of
specialized tests dedicated to ensure the installation and
the upgrade correctness of s3gw the in the Kubernetes cluster.

#### Tag based triggered tests

With a specific Tag pattern, you can trigger a specific
test workflow the involves:

- The **PREVIOUS** `helm charts` version
- The **TARGET** `helm charts` version
- The **PREVIOUS** `image` version
- The **TARGET** `image` version

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

You specify `IMPORT` in the case that the `0.18.0` s3gw's images still
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

## License

Copyright (c) 2023 [SUSE, LLC](http://suse.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
