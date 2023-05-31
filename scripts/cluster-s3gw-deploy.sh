#!/bin/bash
# Copyright © 2021 - 2023 SUSE LLC
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

# IMAGE_TAG is the one built from the 'make build-images'
IMAGE_TAG="$(git describe --tags --always)"

function deploy_s3gw_latest_released {
  helm repo add s3gw https://aquarist-labs.github.io/s3gw-charts
  helm repo update
  helm upgrade --wait --install -n s3gw-acceptance-0 --create-namespace s3gw-0 s3gw/s3gw \
    --set publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set ui.publicDomain="$S3GW_SYSTEM_DOMAIN"
}

function deploy_s3gw_local {
  helm upgrade --install --create-namespace -n s3gw-acceptance-0 \
    --set publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set ui.publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set imageTag="${IMAGE_TAG}" \
    --set ui.imageTag="${IMAGE_TAG}" \
    --set cosi.driver.imageTag="${IMAGE_TAG}" \
    --set cosi.sidecar.imageTag="${IMAGE_TAG}" \
    s3gw-0 ./charts/charts/s3gw --wait "$@"
}

# Ensure we have a value for --system-domain
prepare_system_domain

echo "Deploying s3gw-0"
# Deploy s3gw latest release to test upgrade
if [[ $S3GW_RELEASED ]]; then
  echo "Deploying latest released s3gw charts"
  deploy_s3gw_latest_released
else
  echo "Deploying local s3gw charts"
  deploy_s3gw_local
fi

echo
echo "Done deploying s3gw-0! ✔️"
