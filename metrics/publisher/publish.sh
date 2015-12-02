#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KUBEMARK_LOG_GCLOUD_LOC=$1
KUBEMARK_LOCAL_FILE="kubemark-log.txt"

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
	gsutil cp "${KUBEMARK_LOG_GCLOUD_LOC}" "${KUBEMARK_LOCAL_FILE}"
	# Generate reports (plots) and publish them
	logplot -f "${KUBEMARK_LOCAL_FILE}"
	echo "All files:" $(ls *)
popd

./publish_gcloud_storage.sh "${TEMP}"

