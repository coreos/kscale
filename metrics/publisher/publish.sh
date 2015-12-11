#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: ${KUBEMARK_LOG_FILE:?"Need to set KUBEMARK_LOG_FILE"}
: ${KUBEMARK_PROJECT_NAME:?"Need to set KUBEMARK_PROJECT_NAME"}
: ${OUTPUT_ENV_FILE:?"Need to set OUTPUT_ENV_FILE"}

upload_to_gcs() {
  copy_dir=$1
  bucket_name="metrics-kscale"
  date_format=$(date +"%Y-%m-%d")
  GCS_DIR="${bucket_name}/results/${date_format}/"

  echo "GCS_DIR=\"https://console.developers.google.com/storage/browser/${GCS_DIR}\"" >> "${OUTPUT_ENV_FILE}"
  gsutil cp -r "${copy_dir}" "gs://${GCS_DIR}"
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
