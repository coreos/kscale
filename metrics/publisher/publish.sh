#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KUBEMARK_LOG_FILE=$1
KUBEMARK_REPORT_DIR="kubemark"
KUBEMARK_PROJECT_NAME=${KUBEMARK_PROJECT_NAME:-"default"}

if ! command -v logplot >/dev/null 2>&1; then
  echo "Please install logplot (github.com/coreos/kscale/logplot)"
  exit 1
fi

TEMP=$(mktemp -d 2>/dev/null || mktemp -d -t metrics-publisher.XXXXXX)
cleanup() {
	rm -rf "${TEMP}"
}

trap cleanup EXIT

# Copy kubemark log from cloud
pushd "${TEMP}"
  mkdir -p "${KUBEMARK_REPORT_DIR}/${KUBEMARK_PROJECT_NAME}"
  pushd "${KUBEMARK_REPORT_DIR}/${KUBEMARK_PROJECT_NAME}"
    log_file=$(basename ${KUBEMARK_LOG_FILE})
    cp "${KUBEMARK_LOG_FILE}" "${log_file}"
    # Generate reports (plots) and publish them
    logplot -f "${log_file}"
    echo "kubemark reports:" $(ls *)
  popd
popd

# assumes that helper script are within the same dir
pushd $(dirname $0)
  ./publish_gcloud_storage.sh "${TEMP}"
popd
