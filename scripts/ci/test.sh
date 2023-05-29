#!/bin/bash -e

# build and test spdkcsi, can be invoked manually or by jenkins

DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# shellcheck source=scripts/ci/env
source "${DIR}/env"
# shellcheck source=scripts/ci/common.sh
source "${DIR}/common.sh"

Usage() {
    usage="Uage: $0 [option]
Run all SPDK-CSI tests
Options:
  -h, --help      display this help and exit
      --run-in-vm Run e2e tests in a VM that was prepared by $DIR/preprare.sh.
                  This also runs xPU tests."
    echo "$usage" >&2
}

run_in_vm=false
for arg in $@ ; do
  case "$arg" in
  -h|--help) Usage ; exit ;;
  --run-in-vm) run_in_vm=true ;;
  *) echo "Ignoring unknwon argument: $arg" >&2 ;;
  esac
  shift
done
exit

MYPID=$$

set-timeout "${TIMEOUT_TEST:-60m}" "${MYPID}"

trap on-exit EXIT ERR

#unit_test

vm=""
if $run_in_vm && [[ "${ARCH}" == "amd64" ]]; then
    echo "Running E2E tests in VM"
	vm="vm"
    vm e2e_vm_test
else
    e2e_test
fi

$vm helm_test


if [[ "${ARCH}" == "amd64" ]]; then
    vm_stop
fi
cleanup

exit 0
