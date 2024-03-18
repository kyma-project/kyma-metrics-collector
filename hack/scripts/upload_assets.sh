#!/usr/bin/env bash

# This script has the following argument:
#     - releaseID (mandatory)
#     - packed KMC Chart path name (mandatory)
# ./upload_assets.sh 12345678 kmc-0.0.0.tgz

RELEASE_ID=${1}
KMC_CHART=${2}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   BOT_GITHUB_TOKEN - github token used to upload the asset

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $BOT_GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "::error ::Unable to upload the asset ($filePath): "
    echo "::error ::HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}


UPLOAD_URL="https://uploads.github.com/repos/kyma-project/kyma-metrics-collector/releases/${RELEASE_ID}/assets"

echo -e "\n--- Updating GitHub release ${RELEASE_ID} with ${KMC_CHART} asset"

[[ ! -e ${KMC_CHART} ]] && echo "::error ::Packaged KMC chart does not exist" && exit 1

uploadFile "${KMC_CHART}" "${UPLOAD_URL}?name=${KMC_CHART}"
