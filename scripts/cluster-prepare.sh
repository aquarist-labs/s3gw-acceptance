#!/bin/bash
# Copyright © 2023 SUSE LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

imageS3GW="quay.io/s3gw/s3gw"
imageS3GWUI="quay.io/s3gw/s3gw-ui"
imageCOSIDRIVER="quay.io/s3gw/s3gw-cosi-driver"
imageCOSISIDECAR="quay.io/s3gw/s3gw-cosi-sidecar"

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

source "${SCRIPT_DIR}/helpers.sh"

# UNAME should be darwin or linux
UNAME="$(uname | tr "[:upper:]" "[:lower:]")"

# IMAGE_TAG is the one built from the 'make build-images'
IMAGE_TAG=${IMAGE_TAG:-$(git describe --tags --always)}

#CHARTS VERSION
CHARTS_VER=${CHARTS_VER:-$(helm show chart charts/charts/s3gw | grep version | cut -d':' -f 2 | sed -e 's/^[[:space:]]*//')}
#export CHARTS_VER=$CHARTS_VER

echo "cluster-prepare previous charts-ver  : $CHARTS_VER_PREV"
echo "cluster-prepare current charts-ver   : $CHARTS_VER"
echo "cluster-prepare previous image-tag   : $IMAGE_TAG_PREV"
echo "cluster-prepare current image-tag    : $IMAGE_TAG"

# Ensure we have a value for --system-domain
prepare_system_domain

echo "Preparing k3d environment"

#Install the cert-manager
if ! [[ -v SKIP_CM_INSTALL ]]; then
kubectl create namespace cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager --namespace cert-manager jetstack/cert-manager \
    --set installCRDs=true \
    --set extraArgs[0]=--enable-certificate-owner-ref=true \
    --version 1.10 \
    --wait
fi

#Install COSI resources
if ! [[ -v SKIP_COSI_CRD_INSTALL ]]; then
kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface-api
kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface-controller
fi

# Dump non-static properties used by acceptance tests
dump_suite_properties

echo
echo "Done preparing k3d environment! ✔️"
