#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPORTS_DIR=$1
BUCKET_NAME="metrics-kscale"
DATE_FORMAT=$(date +"%Y-%m-%d")
GCLOUD_STORAGE_FORMAT="gs://${BUCKET_NAME}/kubemark/${DATE_FORMAT}"
pushd "${REPORTS_DIR}"
	gsutil cp * "${GCLOUD_STORAGE_FORMAT}"
popd
