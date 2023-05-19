#!/bin/bash -e

# This common.sh contains all the functions needed for all the e2e tests
# including, configuring proxies, installing packages and tools, building test images, etc.

# FIXME (JingYan): too many "echo"s, try to define a logger function with different logging levels, including
# "info", "warning" and "error", etc. and replace all the "echo" with the logger function

ARCH=$(arch)
if [[ "$(arch)" == "x86_64" ]]; then
	ARCH="amd64"
elif [[ "$(arch)" == "aarch64" ]]; then
	ARCH="arm64"
else
	echo "${ARCH} is not supported"
	exit 1
fi

SPDK_CONTAINER="spdkdev-e2e"
SPDK_SMA_CONTAINER="spdkdev-sma"

vm_qemu_bin=/usr/local/qemu/vfio-user-p3.0/bin/qemu-system-x86_64
vmssh="ssh -p 10000 root@localhost"
vmssh_nonblock="ssh -p 10000 -o StrictHostKeyChecking=accept-new -o ConnectTimeout=1 root@localhost"

export PATH="/var/lib/minikube/binaries/${KUBE_VERSION}:/usr/local/go/bin:${PATH}"

function export_proxy() {
	local http_proxies

	http_proxies=$(env | { grep -Pi "http[s]?_proxy" || true; })
	[ -z "$http_proxies" ] && return 0

	for proxy in $http_proxies; do
		# shellcheck disable=SC2001,SC2005
		echo "$(sed "s/.*=/\U&/" <<< "$proxy")"
		# shellcheck disable=SC2001
		export "$(sed "s/.*=/\U&/" <<< "$proxy")"
	done

	export NO_PROXY="$NO_PROXY,127.0.0.1,localhost,10.0.0.0/8,192.168.0.0/16,.internal"
	export no_proxy="$no_proxy,127.0.0.1,localhost,10.0.0.0/8,192.168.0.0/16,.internal"
}

function check_os() {
	# check distro
	source /etc/os-release
	case $ID in
	fedora)
		distro="fedora"
		;;
	debian)
		echo "Warning: Debian is not officially supported, using Ubuntu setup"
		distro="ubuntu"
		;;
	ubuntu)
		distro="ubuntu"
		;;
	*)
		echo "Only supports Ubuntu and Fedora now"
		exit 1
		;;
	esac
	export distro

	# check nvme-tcp kernel module
	if ! modprobe -n nvme-tcp; then
		echo "failed to load nvme-tcp kernel module"
		echo "upgrade kernel to 5.0+ and install linux-modules-extra package"
		exit 1
	fi
	# check iscsi_tcp kernel module
	if ! modprobe -n iscsi_tcp; then
		echo "failed to load iscsi_tcp kernel module"
		exit 1
	fi
	# check vfio-pci kernel module
	if ! modprobe -n vfio-pci; then
		echo "failed to load vfio-pci kernel module"
		exit 1
	fi
}

# allocate 2048*2M hugepages for prepare_spdk() and prepare_sma()
function allocate_hugepages() {
	local HUGEPAGES_MIN=2048
	local NR_HUGEPAGES=/proc/sys/vm/nr_hugepages
	sync
	echo 3 > /proc/sys/vm/drop_caches
	if [[ -f ${NR_HUGEPAGES} ]]; then
		if [[ $(< ${NR_HUGEPAGES}) -lt ${HUGEPAGES_MIN} ]]; then
			echo ${HUGEPAGES_MIN} > ${NR_HUGEPAGES} || true
		fi
		echo "/proc/sys/vm/nr_hugepages: $(< ${NR_HUGEPAGES})"
		if [[ $(< ${NR_HUGEPAGES}) -lt ${HUGEPAGES_MIN} ]]; then
			echo allocating ${HUGEPAGES_MIN} hugepages failed
			exit 1
		fi
	fi
	cat /proc/meminfo
}

function install_packages_ubuntu() {
	apt-get update -y
	apt-get install -y make \
					gcc \
					curl \
					docker.io \
					conntrack \
					socat \
					wget \
					python3-pip \
					ruby \
					git \
					curl \
					whois \
					cloud-image-utils \
					jq \
					qemu-utils \
					genisoimage \
					netcat
	systemctl start docker

	pip3 install yamllint==1.23.0 shellcheck-py==0.8.0.4
	gem install mdl -v 0.12.0
}

function install_packages_fedora() {
	systemctl stop dnf-makecache.timer || true
	systemctl disable dnf-makecache.timer || true
	systemctl stop dnf-makecache.service || true
	dnf check-update || true
	dnf install -y make \
					gcc \
					curl \
					conntrack \
					bind-utils \
					socat \
					wget \
					python3-pip \
					ruby \
					git \
 					curl \
					mkpasswd \
					cloud-utils \
					jq \
					qemu-img \
					genisoimage \
					netcat
	if ! hash docker &> /dev/null; then
		dnf remove -y docker*
		dnf install -y dnf-plugins-core
		dnf config-manager --add-repo \
			https://download.docker.com/linux/fedora/docker-ce.repo
		dnf check-update || true
		dnf install -y docker-ce docker-ce-cli containerd.io
	fi
	systemctl start docker

	pip3 install yamllint==1.23.0 shellcheck-py==0.8.0.4
	gem install mdl -v 0.12.0
}

function install_golang() {
	if [ -d /usr/local/go ]; then
		golang_info="/usr/local/go already exists, golang install skipped"
		echo "========================================================"
		[ -n "${golang_info}" ] && echo "INFO: ${golang_info}"
		return
	fi
	echo "=============== installing golang ==============="
	GOPKG=go${GOVERSION}.linux-${ARCH}.tar.gz
	curl -s https://dl.google.com/go/"${GOPKG}" | tar -C /usr/local -xzf -
	/usr/local/go/bin/go version
}

function configure_proxy() {
	if [ -n "${DOCKER_MIRROR}" ]; then
		mkdir -p /etc/docker
		cat <<EOF > /etc/docker/daemon.json
{
  "insecure-registries": [
	"${DOCKER_MIRROR}"
  ],
  "registry-mirrors": [
	"https://${DOCKER_MIRROR}"
  ]
}
EOF
	fi
	mkdir -p /etc/systemd/system/docker.service.d
	cat <<- EOF > /etc/systemd/system/docker.service.d/http-proxy.conf
[Service]
Environment="HTTP_PROXY=$HTTP_PROXY"
Environment="HTTPS_PROXY=$HTTPS_PROXY"
Environment="NO_PROXY=$NO_PROXY"
EOF
	systemctl daemon-reload
	systemctl restart docker
	sed -e "s:^\(no_proxy\)=.*:\1=${NO_PROXY}:gI" -i /etc/environment
}

function configure_system_fedora() {
	# Make life easier and set SE Linux to Permissive if it's
	# not already disabled.
	[ "$(getenforce)" != "Disabled" ] && setenforce "Permissive"

	# Disable swap memory so that minikube does not complain.
	# On recent Fedora systemd releases also remove zram tools
	# to keep swap from regenerating.
	if rpm -q --quiet systemd; then
		dnf remove -y zram*
	fi
	swapoff -a

	# check if open-iscsi is installed on host
	iscsi_check_cmd="rpm --quiet -q iscsi-initiator-utils"
	iscsi_remove_cmd="dnf remove -y iscsi-initiator-utils"
	if $iscsi_check_cmd; then
		$iscsi_remove_cmd || true
	fi
}

function setup_cri_dockerd() {
	# use the cri-dockerd adapter to integrate Docker Engine with Kubernetes 1.24 or higher version
	local STATUS
	STATUS="$(systemctl is-active cri-docker.service || true)"
	if [ "${STATUS}" == "active" ]; then
		cri_dockerd_info="cri-docker is already active, cri-dockerd setup skipped"
		echo "========================================================"
		[ -n "${cri_dockerd_info}" ] && echo "INFO: ${cri_dockerd_info}"
		return
	fi

	echo "=============== setting up cri-dockerd ==============="
	echo "=== downloading cri-dockerd-${CRIDOCKERD_VERSION}"
	wget -c https://github.com/Mirantis/cri-dockerd/releases/download/v"${CRIDOCKERD_VERSION}"/cri-dockerd-"${CRIDOCKERD_VERSION}"."${ARCH}".tgz -O - | tar -xz -C /usr/local/bin/ --strip-components 1
	wget https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.service -P /etc/systemd/system/
	wget https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.socket -P /etc/systemd/system/

	# start cri-docker service
	sed -i -e 's,/usr/bin/cri-dockerd,/usr/local/bin/cri-dockerd,' /etc/systemd/system/cri-docker.service
	systemctl daemon-reload
	systemctl enable cri-docker.service
	systemctl enable --now cri-docker.socket

	echo "=== downloading crictl-${CRITOOLS_VERSION}"
	wget -c https://github.com/kubernetes-sigs/cri-tools/releases/download/"${CRITOOLS_VERSION}"/crictl-"${CRITOOLS_VERSION}"-linux-"${ARCH}".tar.gz -O - | tar -xz -C /usr/local/bin/
}

function setup_cni_networking() {
	echo "=============== setting up CNI networking ==============="
	echo "=== downloading 10-crio-bridge.conf and CNI plugins"
	mkdir -p /etc/cni/net.d
	wget https://raw.githubusercontent.com/cri-o/cri-o/v1.23.4/contrib/cni/10-crio-bridge.conf -P /etc/cni/net.d/
	mkdir -p /opt/cni/bin
	wget -c https://github.com/containernetworking/plugins/releases/download/"${CNIPLUGIN_VERSION}"/cni-plugins-linux-"${ARCH}"-"${CNIPLUGIN_VERSION}".tgz -O - | tar -xz -C /opt/cni/bin
}

function stop_host_iscsid() {
	local STATUS
	STATUS="$(systemctl is-enabled iscsid.service >&/dev/null || true)"
	if [ "${STATUS}" == "enabled" ]; then
		systemctl disable iscsid.service
		systemctl disable iscsid.socket
	fi

	STATUS="$(systemctl is-active iscsid.service >&/dev/null || true)"
	if [ "${STATUS}" == "active" ]; then
		systemctl stop iscsid.service
		systemctl stop iscsid.socket
	fi
}

function docker_login {
	if [[ -n "$DOCKERHUB_USER" ]] && [[ -n "$DOCKERHUB_SECRET" ]]; then
		docker login --username "$DOCKERHUB_USER" \
			--password-stdin <<< "$(cat "$DOCKERHUB_SECRET")"
	fi
}

function build_spdkimage() {
	if docker inspect --type=image "${SPDKIMAGE}" >/dev/null 2>&1; then
		spdkimage_info="${SPDKIMAGE} image exists, build skipped"
		echo "========================================================"
		[ -n "${spdkimage_info}" ] && echo "INFO: ${spdkimage_info}"
		return
	fi

	if [ -n "$HTTP_PROXY" ] && [ -n "$HTTPS_PROXY" ]; then
		docker_proxy_opt=("--build-arg" "http_proxy=$HTTP_PROXY" "--build-arg" "https_proxy=$HTTPS_PROXY")
	fi

	echo "============= building spdk container =============="
	spdkdir="${ROOTDIR}/deploy/spdk"
	docker build -t "${SPDKIMAGE}" -f "${spdkdir}/Dockerfile" \
	"${docker_proxy_opt[@]}" "${spdkdir}" && spdkimage_info="${SPDKIMAGE} image build successfully."
}

function build_spdkcsi() {
	# comment the following line to prevent error "make: *** No rule to make target 'clean'.  Stop."
	# make clean
	echo "======== build spdkcsi ========"
	make -C "${ROOTDIR}" spdkcsi
	make -C "${ROOTDIR}" lint
	echo "======== build container ========"
	# XXX: should match image name:tag in Makefile
	sudo docker rmi spdkcsi/spdkcsi:canary > /dev/null || :
	sudo --preserve-env=PATH,HOME make -C "${ROOTDIR}" image
}

function prepare_k8s_cluster() {
	echo "======== prepare k8s cluster with minikube ========"
	sudo modprobe iscsi_tcp
	sudo modprobe nvme-tcp
	sudo modprobe vfio-pci
	export KUBE_VERSION MINIKUBE_VERSION
	sudo --preserve-env HOME="$HOME" "${ROOTDIR}/scripts/minikube.sh" up
}

# FIXME (JingYan): after starting the container, instead of waiting for a fixed number of seconds before executing commands
# in the container in the prepare_spdk() and prepare_sma() functions, we could try to do docker exec here and call spdk's rpc.py
# to try communicating with target. See https://github.com/spdk/spdk/blob/master/test/common/autotest_common.sh#L785

function prepare_spdk() {
	echo "======== start spdk target for storage node ========"
	grep Huge /proc/meminfo
	# start spdk target for storage node
	sudo docker run -id --name "${SPDK_CONTAINER}" --privileged --net host -v /dev/hugepages:/dev/hugepages -v /dev/shm:/dev/shm "${SPDKIMAGE}" /root/spdk/build/bin/spdk_tgt
	sleep 20s
	# wait for spdk target ready
	sudo docker exec -i "${SPDK_CONTAINER}" timeout 5s /root/spdk/scripts/rpc.py framework_wait_init
	# create 1G malloc bdev
	sudo docker exec -i "${SPDK_CONTAINER}" /root/spdk/scripts/rpc.py bdev_malloc_create -b Malloc0 1024 4096
	# create lvstore
	sudo docker exec -i "${SPDK_CONTAINER}" /root/spdk/scripts/rpc.py bdev_lvol_create_lvstore Malloc0 lvs0
	# start jsonrpc http proxy
	sudo docker exec -id "${SPDK_CONTAINER}" /root/spdk/scripts/rpc_http_proxy.py "${JSONRPC_IP}" "${JSONRPC_PORT}" "${JSONRPC_USER}" "${JSONRPC_PASS}"
	sleep 10s
}

function prepare_sma() {
	echo "======== start spdk target for IPU node ========"
	# start spdk target for IPU node
	sudo docker run -id --name "${SPDK_SMA_CONTAINER}" --privileged --net host -v /dev/hugepages:/dev/hugepages -v /dev/shm:/dev/shm -v /var/tmp:/var/tmp -v /lib/modules:/lib/modules "${SPDKIMAGE}"
	sudo docker exec -i "${SPDK_SMA_CONTAINER}" sh -c "HUGEMEM=2048 /root/spdk/scripts/setup.sh; /root/spdk/build/bin/spdk_tgt -S /var/tmp -m 0x3 &"
	sleep 20s
	echo "======== start sma server ========"
	# start sma server
	sudo docker exec -d "${SPDK_SMA_CONTAINER}" sh -c "/root/spdk/scripts/sma.py --config /root/sma.yaml"
	sleep 10s
}

function unit_test() {
	echo "======== run unit test ========"
	make -C "${ROOTDIR}" test
}

function e2e_test() {
	local xpu="$1"
	echo "${xpu}"
	echo "======== run E2E test ========"
	export PATH="/var/lib/minikube/binaries/${KUBE_VERSION}:${PATH}"
	make -C "${ROOTDIR}" e2e-test "${xpu}"
}

function helm_test() {
	sudo docker rm -f "${SPDK_CONTAINER}" > /dev/null || :
	sudo docker rm -f "${SPDK_SMA_CONTAINER}" > /dev/null || :
	make -C "${ROOTDIR}" helm-test
}

function cleanup() {
	sudo docker stop "${SPDK_CONTAINER}" || :
	sudo docker rm -f "${SPDK_CONTAINER}" > /dev/null || :
	sudo docker stop "${SPDK_SMA_CONTAINER}" || :
	sudo docker rm -f "${SPDK_SMA_CONTAINER}" > /dev/null || :
	sudo --preserve-env HOME="$HOME" "${ROOTDIR}/scripts/minikube.sh" clean || :
	# TODO: remove dangling nvmf,iscsi disks
}

# set-timeout <TIMESPEC> <PID> forks a child process that will send
# TERM signal to PID after TIMESPEC has elapsed, unless the child
# process or the "sleep <TIMESPEC>" process have been terminated
# before that.
function set-timeout() {
	local timespec="$1"
	local pid="$2"
	echo "set-timeout: will call 'on-timeout ${pid}' after ${timespec}."
	(on-timeout "${timespec}" "${pid}" || true) &
}

function on-timeout() {
	sleep "$1" >&/dev/null
	# Print information that may reveal reasons for the timeout
	echo "on-timeout, terminate $2"
	dump-debug-info || true
	kill "$2" # stop execution, trigger calling on-exit()
	exit 1
}

# on-exit cleans up all child processes that otherwise might block the
# tes script from terminating. Furthermore, if exit takes place for
# any other reason than explicit "exit" command, backtrace will be
# printed. This happens, for instance, if a command has failed (as we
# are running with "bash -e"), or if an internal timeout has occured.
function on-exit() {
	local exit_status=$?
	if [[ "$BASH_COMMAND" != "exit "* ]] && [[ -z "$backtrace_printed" ]]; then
		# Unexpected exit, possibly due to an error or timeout
		echo "Unexpected error when running: $BASH_COMMAND"
		print_backtrace
		backtrace_printed=1
	fi
	# Always cleanup child processes to prevent getting stuck on exit
	pkill --parent "${MYPID}" || true
	pkill -9 -f -- 'ssh -p 10000 root@localhost' 2>/dev/null || true
	if [ -f /tmp/qemu-vm/qemu.pid ]; then
		kill "$(< /tmp/qemu-vm/qemu.pid)" 2>/dev/null || true
	fi
	cleanup || true
	exit $exit_status
}

function dump-debug-info() {
	echo "===== dump-debug-info: host: ps axf ====="
	ps axf
	echo "===== dump-debug-info: vm: ps axf ====="
	$vmssh_nonblock "ps axf"
	echo "===== end of dump-debug-info ====="
}

function print_backtrace() {
	# if errexit is not enabled, don't print a backtrace
	[[ "$-" =~ e ]] || return 0
	local args=("${BASH_ARGV[@]}")
	# Reset IFS in case we were called from an environment where it was modified
	IFS=" "$'\t'$'\n'
	echo "========== Backtrace start: =========="
	echo ""
	echo "Command: ${BASH_SOURCE[2]}:${LINENO[2]} \"${BASH_COMMAND}\""
	echo ""
	for ((i = 2; i < ${#FUNCNAME[@]}; i++)); do
		local func="${FUNCNAME[$i]}"
		local line_nr="${BASH_LINENO[$((i - 1))]}"
		local src="${BASH_SOURCE[$i]}"
		local bt="" cmdline=()
		if [[ -f "$src" ]]; then
			bt=$(nl -w 4 -ba -nln "$src" | grep -B 5 -A 5 "^${line_nr}[^0-9]" \
					 | sed "s/^/   /g" | sed "s/^   $line_nr /=> $line_nr /g")
		fi
		# If extdebug set the BASH_ARGC[i], try to fetch all the args
		if ((BASH_ARGC[i] > 0)); then
			# Use argc as index to reverse the stack
			local argc=${BASH_ARGC[i]} arg
			for arg in "${args[@]::BASH_ARGC[i]}"; do
				cmdline[argc--]="[\"$arg\"]"
			done
			args=("${args[@]:BASH_ARGC[i]}")
		fi
		echo "in $src:$line_nr -> $func($(
						IFS=","
						printf '%s\n' "${cmdline[*]:-[]}"
				))"
		echo "     ..."
		echo "${bt:-backtrace unavailable}"
		echo "     ..."
	done
	echo ""
	echo "========== Backtrace end =========="
	return 0
}

function vm_build() {
	# build oracle qemu
	[ -f "$vm_qemu_bin" ] && {
		echo "vm-build: already built: $vm_qemu_bin"
		return 0
	}
	[ -d "${ROOTDIR}"/../spdk ] || git clone https://github.com/spdk/spdk "${ROOTDIR}"/../spdk

	"${ROOTDIR}"/../spdk/test/common/config/vm_setup.sh -i -u -t qemu
}

function vm_start() {
	if [ ! -f "$vm_qemu_bin" ]; then
		echo "$vm_qemu_bin does not exist."
		exit 1
	fi

	if ! $vmssh_nonblock "true"; then
		__vm_qemu_launch || {
			echo "vm_qemu_launch failed"
			return 1
		}
	fi

	# Configure proxies
	local filep
	local linep
	local file
	vars=("filep='' linep='' file='/etc/environment'")
	vars+=("filep='' linep='export ' file='/etc/profile.d/proxy.sh'")
	vars+=("filep='[Service]' linep='Environment=' file='/etc/systemd/system/containerd.service.d/proxy.conf'")
	for var in "${vars[@]}"
	do
		(
			eval "$var"
			ext_no_proxy=$no_proxy,localhost,127.0.0.1,.internal,10.0.0.0/8,192.168.0.0/16
			cat <<EOF |
${filep}
${linep}http_proxy=${http_proxy:-}
${linep}https_proxy=${https_proxy:-}
${linep}no_proxy=$ext_no_proxy
${linep}HTTP_PROXY=${HTTP_PROXY:-}
${linep}HTTPS_PROXY=${HTTPS_PROXY:-}
${linep}NO_PROXY=$ext_no_proxy

EOF
			$vmssh "mkdir -p $(dirname "$file"); cat > $file"
		)
	done

	# Copy scripts to vm. After this the vm function allows
	# calling common functions in vm, too.
	file_name=$(basename "$(realpath "${ROOTDIR}")")
	tar cz --exclude '*.git*' "${ROOTDIR}"/../"${file_name}" | $vmssh "tar xzf -"

	# Set port forwards from VM to local host.
	$vmssh -R 9009:localhost:9009 -R 4420:localhost:4420 -R 3260:localhost:3260 -R 4421:localhost:4421 -R 5114:localhost:5114 "sleep inf" &
}

function vm_stop() {
	echo "shutting down qemu"
	$vmssh "shutdown -h now"
}

function __vm_qemu_launch() {
	# Prepare the image and cloud-init
	local workerdir
	local fedora_qcow2
	local cloudisodir
	local cloud_iso
	local qemu

	workerdir=/tmp/qemu-vm
	fedora_qcow2=${workerdir}/fedora-cloud-base.qcow2
	cloudisodir=${workerdir}/cloud-init-iso-root
	cloud_iso=${workerdir}/seed.iso
	qemu=${QEMU:-/usr/local/qemu/vfio-user-p3.0/bin/qemu-system-x86_64}

	mkdir -p "$(dirname ${fedora_qcow2})"
	mkdir -p "${cloudisodir}"

	for required_cmd in curl mkpasswd cloud-localds jq "${qemu}"; do
		command -v "${required_cmd}" >/dev/null || {
			echo "missing: ${required_cmd}"
			exit 1
		}
	done

	curl -Lk https://download.fedoraproject.org/pub/fedora/linux/releases/37/Cloud/x86_64/images/Fedora-Cloud-Base-37-1.7.x86_64.qcow2 > "${fedora_qcow2}"
	echo "Check disk info and resize the qemu VM img"
	df -h
	qemu-img resize "${fedora_qcow2}" +20G
	[ -f ~/.ssh/id_rsa ] || ssh-keygen -t rsa -f ~/.ssh/id_rsa -P ''

	# Prepare cloud-init
	(
		cd "${cloudisodir}" || exit
		echo "instance-id: qemu-vm" > meta_data
		echo "local-hostname: qemu-vm" >> meta_data
		cat > user_data << EOF
#cloud-config
disable_root: False
chpasswd: { expire: False }
ssh_pwauth: True
users:
- name: root
  lock_passwd: False
  ssh_authorized_keys:
  - $(< ~/.ssh/id_rsa.pub)
- name: fedora
  lock_passwd: False
  passwd: "$(echo fedora | mkpasswd -s)"
  ssh_authorized_keys:
  - $(< ~/.ssh/id_rsa.pub)
chpasswd:
  expire: False
  users:
  - name: root
    password: "$(echo root | mkpasswd -s)"
runcmd:
  - [ mkdir, -p, /etc/default ]
  - [ touch, /etc/default/grub ]
  - [ sh, -c, "echo 'GRUB_CMDLINE_LINUX_DEFAULT=\"\${GRUB_CMDLINE_LINUX_DEFAULT} scsi_mod.use_blk_mq=1\"' >> /etc/default/grub" ]
  - [ grub2-mkconfig, -o, /boot/grub2/grub.cfg ]
  - [ reboot ]
EOF
		cloud-localds "${cloud_iso}" user_data meta_data
	)
	[ -f "${cloud_iso}" ] || {
		echo "failed to create cloud-init image ${cloud_iso}"
		exit 1
	}

	echo "checking current RAM info"
	free -m
	echo "clear cache"
	sync; echo 3 > /proc/sys/vm/drop_caches
	echo "checking current RAM info"
	free -m
	echo "setting the hugepage"
	grep Huge /proc/meminfo
	rm -f ${workerdir}/*.log

	qemu_launch_cmd="sudo ${qemu} \
				-m 6144 --enable-kvm -cpu host \
				-object memory-backend-file,id=mem,size=6144M,mem-path=/dev/shm,share=on,prealloc=yes,host-nodes=0,policy=bind \
				-numa node,memdev=mem \
				-smp 6 \
				-serial file:${workerdir}/serial.log -D ${workerdir}/qemu.log \
				-chardev file,path=${workerdir}/seabios.log,id=seabios \
				-device isa-debugcon,iobase=0x402,chardev=seabios \
				-net user,hostfwd=tcp::10000-:22,hostfwd=tcp::10001-:8765 -net nic \
				-drive file=${fedora_qcow2},if=none,id=os_disk \
				-device ide-hd,drive=os_disk,bootindex=0 \
				-device virtio-scsi-pci,num_queues=2 \
				-device scsi-hd,drive=hd,vendor=RAWSCSI \
				-drive if=none,id=hd,file=${cloud_iso},format=raw \
				-qmp tcp:localhost:9090,server,nowait -device pci-bridge,chassis_nr=1,id=pci.spdk.0 \
				-device pci-bridge,chassis_nr=2,id=pci.spdk.1 \
				-device pci-bridge,chassis_nr=3,id=pci.spdk.2 \
				-device pci-bridge,chassis_nr=4,id=pci.spdk.3"
	echo "$qemu_launch_cmd" > "${workerdir}/qemu.launch.sh"
	set -x
	$qemu_launch_cmd &
	qemu_pid=$!
	set +x
	echo "$qemu_pid" >"${workerdir}/qemu.pid"
	sleep 1

	if [ -d "/proc/$qemu_pid" ]; then
		echo "VM started successfully"
	else
		echo "VM failed to start"
		exit 1
	fi

	# Now the virtual machine is booting up. Wait for ssh to start working
	if [ ! -f ~/.ssh/known_hosts ]; then
		touch ~/.ssh/known_hosts
	fi

	ssh-keygen -R "[localhost]:10000"

	echo "waiting for cloud-init to finish"
	a=0
	while ((a++ < 120))
	do
		$vmssh_nonblock 'cloud-init status --wait' 2>/dev/null && break
		sleep 1
		echo -n "."
	done
}

function vm() {
	$vmssh "DIR=spdk-csi/scripts/ci; source \${DIR}/env; source \${DIR}/common.sh; export_proxy; distro=fedora; $*"
}