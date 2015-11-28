#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

source "./hack/lib/util.sh"
source "./hack/lib/logging.sh"
source "./hack/lib/etcd.sh"

cleanup() {
  kube::etcd::cleanup
  kube::log::status "Integration test cleanup complete"
}

trap cleanup EXIT

kube::etcd::start
kube::log::status "Start benchmarking..."
go test -v -bench . -run xxxx