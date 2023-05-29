#!/bin/bash -e

DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# shellcheck source=scripts/ci/env
source "${DIR}/env"
# shellcheck source=scripts/ci/common.sh
source "${DIR}/common.sh"

MYPID=$$

set-timeout "${TIMEOUT_PREPARE:-45m}" "${MYPID}"

trap on-exit EXIT ERR

PROMPT_FLAG=${PROMPT_FLAG:-true}

if [[ $(id -u) != "0" ]]; then
	echo "Go away user, come back as root."
	exit 1
fi

while getopts 'yu:p:vm' optchar; do
	case "$optchar" in
		y)
			PROMPT_FLAG=false
			;;
		u)
			DOCKERHUB_USER="$OPTARG"
			;;
		p)
			DOCKERHUB_SECRET="$OPTARG"
			;;
		v) PREPARE_VM=true
		   ;;
		*)
			echo "$0: invalid argument '$optchar'"
			exit 1
			;;
	esac
done

if $PROMPT_FLAG; then
	echo "This script is meant to run on CI nodes."
	echo "It will install packages and docker images on current host."
	echo "Make sure you understand what it does before going on."
	read -r -p "Do you want to continue (yes/no)? " yn
	case "${yn}" in
		y|Y|yes|Yes|YES) :;;
		*) exit 0;;
	esac
fi

export_proxy
build_spdkimage
build_spdkcsi
allocate_hugepages
prepare_spdk
prepare_sma

vm=
# build oracle qemu for nvme
if $PREPARE_VM && [[ "${ARCH}" == "amd64" ]]; then
	vm_build
	vm_start
	vm="vm"
fi
$vm check_os
$vm install_packages_"${distro}"
$vm install_golang
$vm configure_proxy
[ "${distro}" == "fedora" ] && $vm configure_system_fedora
$vm setup_cri_dockerd
$vm setup_cni_networking
$vm stop_host_iscsid
$vm docker_login
# workaround minikube permissions issues when running as root in ci(-like) env
$vm sysctl fs.protected_regular=0
$vm prepare_k8s_cluster

exit 0
