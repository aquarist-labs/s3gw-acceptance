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

# UNAME should be darwin or linux
UNAME="$(uname | tr "[:upper:]" "[:lower:]")"

# IMAGE_TAG is the one built from the 'make build-images'
IMAGE_TAG="$(git describe --tags --always)"

function check_dependency {
	for dep in "$@"
	do
		if ! [ -x "$(command -v $dep)" ]; then
			echo "Error: ${dep} is not installed." >&2
  			exit 1
		fi
	done

}

function create_docker_pull_secret {
	if [[ "$REGISTRY_USERNAME" != "" && "$REGISTRY_PASSWORD" != "" && ! $(kubectl get secret regcred > /dev/null 2>&1) ]];
	then
		kubectl create secret docker-registry regcred \
			--docker-server https://index.docker.io/v1/ \
			--docker-username $REGISTRY_USERNAME \
			--docker-password $REGISTRY_PASSWORD
	fi
}

function retry {
  retry=0
  maxRetries=$1
  retryInterval=$2
  local result
  until [ ${retry} -ge ${maxRetries} ]
  do
    echo -n "."
    result=$(eval "$3") && break
    retry=$[${retry}+1]
    sleep ${retryInterval}
  done

  if [ ${retry} -ge ${maxRetries} ]; then
    echo "Failed to run "$3" after ${maxRetries} attempts!"
    exit 1
  fi

  echo " ✔️"
}

function deploy_s3gw_latest_released {
  helm repo add s3gw https://aquarist-labs.github.io/s3gw-charts
  helm repo update
  helm upgrade --wait --install -n s3gw-acceptance --create-namespace s3gw s3gw/s3gw \
    --set publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set ui.publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set serviceName=s3gw \
    --set ui.serviceName=s3gw-ui \
    --set cosi.enabled="true"
}

# Ensure we have a value for --system-domain
prepare_system_domain
# Create docker registry image pull secret
create_docker_pull_secret

echo "Installing s3gw"
# Deploy s3gw latest release to test upgrade
if [[ $S3GW_RELEASED ]]; then
  echo "Deploying latest released s3gw images"
  deploy_s3gw_latest_released
else
  echo "Importing locally built s3gw images"
  k3d image import -c s3gw-acceptance "${imageS3GW}:${IMAGE_TAG}"
  echo "Importing locally built s3gw image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageS3GWUI}:${IMAGE_TAG}"
  echo "Importing locally built s3gw-ui image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageCOSIDRIVER}:${IMAGE_TAG}"
  echo "Importing locally built s3gw-cosi-driver image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageCOSISIDECAR}:${IMAGE_TAG}"
  echo "Importing locally built s3gw-cosi-sidecar image Completed ✔️"

  helm upgrade --install --create-namespace -n s3gw-acceptance \
    --set publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set ui.publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set serviceName=s3gw \
    --set ui.serviceName=s3gw-ui \
    --set cosi.enabled="true" \
    --set imageTag="${IMAGE_TAG}" \
    --set ui.imageTag="${IMAGE_TAG}" \
    --set cosi.driver.imageTag="${IMAGE_TAG}" \
    --set cosi.sidecar.imageTag="${IMAGE_TAG}" \
    s3gw ./charts/charts/s3gw --wait "$@"

fi

echo
echo "Done preparing k3d environment!"
