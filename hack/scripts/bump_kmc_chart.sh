#!/usr/bin/env bash

# This script bumps the KMC images in the chart, utils and the KMC chart version.
# It has the following arguments:
#   - release tag (mandatory)
# ./bump_kmc_chart.sh 0.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


RELEASE_TAG=$1
VALUES_YAML="resources/kyma-metrics-collector/values.yaml"

KEYS=$(yq e '.global.images | keys | .[]' $VALUES_YAML | grep 'kyma_metrics')

# bump images in resources/kyma-metrics-collector/values.yaml
for key in $KEYS
do
    # yq removes empty lines when editing files, so the diff and patch are used to preserve formatting.
    yq e ".global.images.$key.version = \"$RELEASE_TAG\"" $VALUES_YAML > $VALUES_YAML.new
    yq '.' $VALUES_YAML > $VALUES_YAML.noblanks
    diff -B $VALUES_YAML.noblanks $VALUES_YAML.new > resources/kyma-metrics-collector/patch.file
    patch $VALUES_YAML resources/kyma-metrics-collector/patch.file
    rm $VALUES_YAML.noblanks
    rm resources/kyma-metrics-collector/patch.file
    rm $VALUES_YAML.new
done

yq e ".version = \"$RELEASE_TAG\"" -i resources/kyma-metrics-collector/Chart.yaml
yq e ".appVersion = \"$RELEASE_TAG\"" -i resources/kyma-metrics-collector/Chart.yaml
