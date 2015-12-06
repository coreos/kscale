#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export KUBE_SKIP_UPDATE="y"
export KUBE_PROMPT_FOR_UPDATE="N"

: ${ENV_SCRIPT_DIR:?"Need to set ENV_SCRIPT_DIR"}
: ${PUBLISHER_DIR:?"Need to set PUBLISHER_DIR"}
# Assumes that most of the time we run nightly builds in k8s repo
K8S_DIR=${K8S_DIR:-`pwd`}
KUBEMARK_LOG_DIR=${KUBEMARK_LOG_DIR:-"/var/log"}

cleanup() {
  (source "${ENV_SCRIPT_DIR}/kubemark-env.sh" && ${K8S_DIR}/test/kubemark/stop-kubemark.sh) || true
  (source "${ENV_SCRIPT_DIR}/k8s-cluster-env.sh" && ${K8S_DIR}/cluster/kube-down.sh) || true
}

trap cleanup EXIT

source "${ENV_SCRIPT_DIR}/k8s-cluster-env.sh" && ${K8S_DIR}/cluster/kube-up.sh

source "${ENV_SCRIPT_DIR}/kubemark-env.sh" && ${K8S_DIR}/test/kubemark/start-kubemark.sh
# Even if test failed, the log file is still useful to us
# TODO: if test failed, change log file name to "kubemark-log-failed.txt"
${K8S_DIR}/test/kubemark/run-e2e-tests.sh \
  --ginkgo.focus="should\sallow\sstarting\s30\spods\sper\snode" \
  --delete-namespace="false" --gather-resource-usage="false" \
  | tee "${KUBEMARK_LOG_DIR}/kubemark-log.txt" \
  || true


${PUBLISHER_DIR}/publish.sh "${KUBEMARK_LOG_DIR}/kubemark-log.txt"
