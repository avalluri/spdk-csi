#!/bin/bash -e

# build and test spdkcsi, can be invoked manually or by jenkins

DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# shellcheck source=scripts/ci/env
source "${DIR}/env"
# shellcheck source=scripts/ci/common.sh
source "${DIR}/common.sh"

MYPID=$$

set-timeout "${TIMEOUT_TEST:-60m}" "${MYPID}"

trap on-exit EXIT ERR

export_proxy
docker_login
build_spdkcsi
prepare_k8s_cluster
prepare_spdk
prepare_sma
unit_test
e2e_test xpu=false
helm_test
cleanup


# Prepare VM for nvme e2e test for x86_64
if [[ "${ARCH}" == "amd64" ]]; then
    vm_start
    vm check_os
    vm "install_packages_\${distro}" # expands $distro on vm, not here
    vm install_golang
    vm configure_proxy
    vm "configure_system_\${distro}"
    vm setup_cri_dockerd
    vm setup_cni_networking
    vm stop_host_iscsid
    vm docker_login

    prepare_spdk
    prepare_sma

    vm prepare_k8s_cluster
    vm build_spdkcsi
    vm e2e_test xpu=true

    vm_stop
    cleanup
fi

exit 0
