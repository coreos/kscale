#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script takes a directory as input.
# It uploads all files within that directory into gcloud storage.
# The cloud storage directory is defined as $GCLOUD_STORAGE_FORMAT

REPORTS_DIR=$1
BUCKET_NAME="metrics-kscale"
DATE_FORMAT=$(date +"%Y-%m-%d")
RESULT_DIR_LAYOUT="${BUCKET_NAME}/results/${DATE_FORMAT}/"

pushd "${REPORTS_DIR}"
	echo "uploading files to https://console.developers.google.com/storage/browser/${RESULT_DIR_LAYOUT}"
	gsutil cp -r ./ "gs://${RESULT_DIR_LAYOUT}"
popd
