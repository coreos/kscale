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
GCLOUD_STORAGE_FORMAT="gs://${BUCKET_NAME}/results/${DATE_FORMAT}/"

pushd "${REPORTS_DIR}"
	echo "uploading files to ${GCLOUD_STORAGE_FORMAT}"
	gsutil cp -r ./ "${GCLOUD_STORAGE_FORMAT}"
popd
