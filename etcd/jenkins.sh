#!/usr/bin/env bash

set -o errexit
#set -o nounset
set -o pipefail

# many assumptions made here to set up jenkins environment
source $HOME/.bash_profile

export CLOUDSDK_CORE_DISABLE_PROMPTS=1
# gcloud config set core/disable_prompts 1

export KUBE_SKIP_CONFIRMATIONS=y
export KUBE_FASTBUILD=true
export KUBE_GCE_NETWORK=default
# export NUM_NODES="6"
export MASTER_SIZE="n1-standard-8"
export NODE_SIZE="n1-standard-2"
export KUBE_GCE_INSTANCE_PREFIX="jenkinse2e"

gcloud version
gsutil version
go version
env

# cleanup() {
# }
# trap cleanup EXIT

build_k8s() {
  make clean
  go run ./hack/e2e.go -v --build
}

upload_gcs() {
  bucket_name="e2e-etcd3"
  date_format=$(date +"%Y-%m-%d")
  GCS_DIR=${GCS_DIR:-"${bucket_name}/jenkins/${date_format}/${BUILD_NUMBER}"}
  ls -l _artifacts/
  gsutil cp -r e2e.log _artifacts "gs://${GCS_DIR}"
  GCS_HTTP_LOC="https://console.developers.google.com/storage/browser/${GCS_DIR}"
  echo "Uploaded:", $GCS_HTTP_LOC
}

run_e2e() {
  go run ./hack/e2e.go -v --up
  # No matter what each command ends up, we should delete the cluster
  local -r ginkgo_test_args="${GINKGO_TEST_ARGS}"
  (go run ./hack/e2e.go -v --test \
    ${ginkgo_test_args:+--test_args="${ginkgo_test_args}"} | tee e2e.log) \
    && exitcode=0 || exitcode=$?
  rm -rf _artifacts
  ./cluster/log-dump.sh
  upload_gcs || true
  go run ./hack/e2e.go -v --down
  return $exitcode
}

if ! build_k8s; then
  MAIL_SUBJECT="[k8s-etcd jenkins] Failed ${BUILD_TAG}" MAIL_TEXT="Build failed!" "$HOME/mail/mailgun.sh" 
  exit 1
fi

if run_e2e; then
  MAIL_TEXT=$(printf "Jenkins URL: ${BUILD_URL}\nGCS: ${GCS_HTTP_LOC}") \
  MAIL_SUBJECT="[k8s-etcd jenkins] Passed ${BUILD_TAG}" "$HOME/mail/mailgun.sh" 
else
  MAIL_TEXT=$(printf "Jenkins URL: ${BUILD_URL}\nGCS: ${GCS_HTTP_LOC}") \
  MAIL_SUBJECT="[k8s-etcd jenkins] Failed ${BUILD_TAG}" "$HOME/mail/mailgun.sh" 
  exit 1
fi

exit 0