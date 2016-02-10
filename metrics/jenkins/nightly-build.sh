#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export KUBE_SKIP_UPDATE="y"
export KUBE_PROMPT_FOR_UPDATE="N"

: ${INPUT_ENV_DIR:?"Need to set INPUT_ENV_DIR"}
: ${PUBLISHER_DIR:?"Need to set PUBLISHER_DIR"}
: ${OUTPUT_ENV_FILE}:?"Need to set OUTPUT_ENV_FILE"}
# Assumes that most of the time we run nightly builds in k8s repo
K8S_DIR=${K8S_DIR:-`pwd`}

# We're going to assume that the output file is reused all the time and use it as
# fast turnaround to record last average running rate.
LAST_AVG_RUNING_RATE=$((cat "${OUTPUT_ENV_FILE}" | grep -E "^avg_running_rate=" | cut -d '=' -f2) || echo "")
echo "" > "${OUTPUT_ENV_FILE}"

# Used to tell if e2e test has been run successfully.
# Note that if e2e test wasn't run, it's seen as a failure.
E2E_TEST_SUCCEED="n"
# Used to tell if test results (incl. plots) has been uploaded successfully.
TEST_RESULTS_UPLOADED="n"

TEMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t metrics-publisher.XXXXXX)

cleanup() {
  rm -rf "${TEMPDIR}"
  (source "${INPUT_ENV_DIR}/kubemark-env.sh" && ${K8S_DIR}/test/kubemark/stop-kubemark.sh) || true
  (source "${INPUT_ENV_DIR}/k8s-cluster-env.sh" && ${K8S_DIR}/cluster/kube-down.sh) || true

  echo "output env:"
  cat "${OUTPUT_ENV_FILE}"

  exitCode="0"
  if [ "x${E2E_TEST_SUCCEED}" != "xy" ]; then
    exitCode="1"
  fi
  if [ "x${TEST_RESULTS_UPLOADED}" != "xy" ]; then
    exitCode="1"
  fi
  exit "${exitCode}"
}

trap cleanup EXIT

source "${INPUT_ENV_DIR}/k8s-cluster-env.sh" && ${K8S_DIR}/cluster/kube-up.sh

source "${INPUT_ENV_DIR}/kubemark-env.sh" && ${K8S_DIR}/test/kubemark/start-kubemark.sh
# If test succeeded, the log file is named "kubemark-log.txt"
# If test succeeded, the log file is named "kubemark-log-fail.txt"
kubemark_log_file="kubemark-log.txt"
("${K8S_DIR}/test/kubemark/run-e2e-tests.sh" --ginkgo.focus="should\sallow\sstarting\s30\spods\sper\snode" --gather-resource-usage="false" \
  | tee "${TEMPDIR}/${kubemark_log_file}") || true

# For some reason, we can't trust e2e test script exit code
if [ "x$(tail -n 1 ${TEMPDIR}/${kubemark_log_file})" == "xTest Suite Passed" ]; then
  E2E_TEST_SUCCEED="y"
fi
echo "E2E_TEST_SUCCEED=\"${E2E_TEST_SUCCEED}\"" >> "${OUTPUT_ENV_FILE}"

if [ -f "${TEMPDIR}/${kubemark_log_file}" ]; then
  if [ "x${E2E_TEST_SUCCEED}" != "xy" ]; then
    fail_kubemark_log_file="kubemark-log-fail.txt"
    mv "${TEMPDIR}/${kubemark_log_file}" "${TEMPDIR}/${fail_kubemark_log_file}"
    kubemark_log_file="${fail_kubemark_log_file}"
  fi

  if source "${INPUT_ENV_DIR}/publisher-env.sh" && \
    PUBLISH_WORK_DIR="${TEMPDIR}" KUBEMARK_LOG_FILE="${TEMPDIR}/${kubemark_log_file}" OUTPUT_ENV_FILE="${OUTPUT_ENV_FILE}" "${PUBLISHER_DIR}/publish.sh"; then
    echo "last_avg_running_rate=${LAST_AVG_RUNING_RATE}" >> "${OUTPUT_ENV_FILE}"
    TEST_RESULTS_UPLOADED="y"
  fi
fi
echo "TEST_RESULTS_UPLOADED=\"${TEST_RESULTS_UPLOADED}\"" >> "${OUTPUT_ENV_FILE}"
