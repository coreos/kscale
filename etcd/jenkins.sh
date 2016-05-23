#!/usr/bin/env bash

# many assumptions made here to set up jenkins environment
source $HOME/.bash_profile

set -o errexit
set -o nounset
set -o pipefail


export CLOUDSDK_CORE_DISABLE_PROMPTS=1
# gcloud config set core/disable_prompts 1

export KUBE_SKIP_CONFIRMATIONS=y
export KUBE_FASTBUILD=true
export KUBE_GCE_NETWORK=default
# export NUM_NODES="6"
# export MASTER_SIZE="n1-standard-16"
# export NODE_SIZE="n1-standard-2"

gcloud version
gsutil version
go version
env

# function cleanup() {
# }
# trap cleanup EXIT

function build_k8s() {
  make clean
  go run ./hack/e2e.go -v --build
}

function upload_gcs() {
  bucket_name="e2e-etcd3"
  date_format=$(date +"%Y-%m-%d")
  GCS_DIR=${GCS_DIR:-"${bucket_name}/jenkins-${JOB_NAME}/${date_format}/${BUILD_NUMBER}"}
  ls -l _artifacts/
  gsutil -m cp -r e2e.log _artifacts "gs://${GCS_DIR}"
  GCS_HTTP_LOC="https://console.developers.google.com/storage/browser/${GCS_DIR}"
  echo "Uploaded: ${GCS_HTTP_LOC}"
}

function run_e2e() {
  local -r ginkgo_test_args="${1}"
  echo "ginkgo_test_args: ${ginkgo_test_args}"
  go run ./hack/e2e.go -v --test \
      ${ginkgo_test_args:+--test_args="${ginkgo_test_args}"}
}

function parse_e2e_result() {
  if [ "$(tail -n 1 e2e.log)" != "Test Suite Passed" ]; then
    return 1
  fi
  return 0
}


function run_default() {
  export GINKGO_PARALLEL="y"
  run_e2e "--ginkgo.skip=\[Slow\]|\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]" | tee e2e.log
  parse_e2e_result || exitcode=1
  run_e2e "--ginkgo.focus=\[Slow\] --ginkgo.skip=\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]" | tee -a e2e.log
  parse_e2e_result || exitcode=1
}

function run_test() {
  go run ./hack/e2e.go -v --up
  # No matter what each command ends up, we should delete the cluster

  if [ "${GINKGO_TEST_ARGS}" == "default" ]; then
    run_default
  else
    run_e2e "${GINKGO_TEST_ARGS:-}" | tee e2e.log
    parse_e2e_result || exitcode=1
  fi

  e2e_result=$(tail -n 6 e2e.log | sed -r "s/\[([0-9]{1,2}(;[0-9]{1,2})*)?m//g")

  rm -rf _artifacts
  ./cluster/log-dump.sh || true
  upload_gcs || true
  go run ./hack/e2e.go -v --down || true
}

if ! build_k8s; then
  MAIL_SUBJECT="${BUILD_TAG} Failed" MAIL_TEXT="Build failed!" "$HOME/mail/mailgun.sh" 
  exit 1
fi

exitcode=0
run_test
export MAIL_TEXT=$(printf "Jenkins URL: ${BUILD_URL}\nGCS: ${GCS_HTTP_LOC}\n\nResult: ${e2e_result}")
if [ "${exitcode}" == "0" ]; then
  MAIL_SUBJECT="${BUILD_TAG} Passed" "$HOME/mail/mailgun.sh" 
else
  MAIL_SUBJECT="${BUILD_TAG} Failed" "$HOME/mail/mailgun.sh" 
  exit 1
fi

exit 0