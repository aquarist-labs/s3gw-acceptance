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

IMAGE_TAG=${IMAGE_TAG:-"$(git describe --tags --always)"}
imageS3GW="quay.io/s3gw/s3gw"
imageS3GWUI="quay.io/s3gw/s3gw-ui"
imageCOSIDRIVER="quay.io/s3gw/s3gw-cosi-driver"
imageCOSISIDECAR="quay.io/s3gw/s3gw-cosi-sidecar"

if ! [[ -v SKIP_IMPORT_IMAGES ]]; then
  echo "--- Building & Importing s3gw images ---"
  (cd cosi-driver; make build)
  docker build -t "${imageCOSIDRIVER}:v${IMAGE_TAG}" -t "${imageCOSIDRIVER}:latest" -f cosi-driver/Dockerfile cosi-driver
  echo "Building s3gw-cosi-driver image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageCOSIDRIVER}:v${IMAGE_TAG}"
  echo "Importing s3gw-cosi-driver image Completed ✔️"
  (cd cosi-sidecar; make build)
  docker build -t "${imageCOSISIDECAR}:v${IMAGE_TAG}" -t "${imageCOSISIDECAR}:latest" -f cosi-sidecar/Dockerfile cosi-sidecar
  echo "Building s3gw-cosi-sidecar image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageCOSISIDECAR}:v${IMAGE_TAG}"
  echo "Importing s3gw-cosi-sidecar image Completed ✔️"
  docker build -t "${imageS3GWUI}:v${IMAGE_TAG}" -t "${imageS3GWUI}:latest" -f ui/src/frontend/Dockerfile ui/src/frontend
  echo "Building s3gw-ui image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageS3GWUI}:v${IMAGE_TAG}"
  echo "Importing s3gw-ui image Completed ✔️"
  docker build -t "${imageS3GW}:v${IMAGE_TAG}" -t "${imageS3GW}:latest" -f dockerfiles/Dockerfile.s3gw .
  echo "Building s3gw image Completed ✔️"
  k3d image import -c s3gw-acceptance "${imageS3GW}:v${IMAGE_TAG}"
  echo "Importing s3gw image Completed ✔️"
fi
