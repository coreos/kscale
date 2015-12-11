#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: ${KUBEMARK_LOG_FILE:?"Need to set KUBEMARK_LOG_FILE"}
: ${KUBEMARK_PROJECT_NAME:?"Need to set KUBEMARK_PROJECT_NAME"}

upload_to_gcs() {
  copy_dir=$1
  bucket_name="metrics-kscale"
  date_format=$(date +"%Y-%m-%d")
  gcs_dir="${bucket_name}/results/${date_format}/"
	echo "uploading files to https://console.developers.google.com/storage/browser/${gcs_dir}"
	gsutil cp -r "${copy_dir}" "gs://${gcs_dir}"
}

if ! command -v logplot >/dev/null 2>&1; then
  echo "Please install logplot (github.com/coreos/kscale/logplot)"
  exit 1
fi

# Current dir is the workspace.
mkdir -p "${KUBEMARK_PROJECT_NAME}"
mv "${KUBEMARK_LOG_FILE}" "${KUBEMARK_PROJECT_NAME}/"
pushd "${KUBEMARK_PROJECT_NAME}"
  log_file=$(basename ${KUBEMARK_LOG_FILE})
  logplot -f "${log_file}"
  echo "kubemark reports:" $(ls *)
popd

upload_to_gcs `pwd`
