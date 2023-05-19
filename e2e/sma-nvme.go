package e2e

import (
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	smaNVMEConfigMapData = `{
  "nodes": [
    {
      "name": "localhost",
      "rpcURL": "http://127.0.0.1:9009",
      "targetType": "nvme-tcp",
      "targetAddr": "127.0.0.1"
    }
  ]
}`
)

var _ = ginkgo.Describe("SPDKCSI-SMA-NVME", func() {
	f := framework.NewDefaultFramework("spdkcsi")
	ginkgo.BeforeEach(func() {
		deployConfigs(smaNVMEConfigMapData)
		deploySmaNvmeConfig()
		deployCsi()
	})

	ginkgo.AfterEach(func() {
		deleteCsi()
		deleteSmaNvmeConfig()
		deleteConfigs()
	})

	ginkgo.Context("Test SPDK CSI SMA NVME", func() {
		ginkgo.It("Test SPDK CSI SMA NVME", func() {
			if !isXpu() {
				ginkgo.Skip("Skipping SPDKCSI-SMA-NVME test: Running inside a virtual machine")
			}

			ginkgo.By("checking controller statefulset is running", func() {
				err := waitForControllerReady(f.ClientSet, 4*time.Minute)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			ginkgo.By("checking node daemonset is running", func() {
				err := waitForNodeServerReady(f.ClientSet, 2*time.Minute)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			ginkgo.By("log verification for SMA grpc connection", func() {
				expLogerrMsgMap := map[string]string{
					"connected to SMA server 127.0.0.1:5114 with TargetType as xpu-sma-nvme": "failed to catch the log about the connection to SMA server from node",
				}
				err := verifyNodeServerLog(expLogerrMsgMap)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			ginkgo.By("create multiple pvcs and a pod with multiple pvcs attached, and check data persistence after the pod is removed and recreated", func() {
				deployMultiPvcs()
				deployTestPodWithMultiPvcs()
				err := waitForTestPodReady(f.ClientSet, 5*time.Minute)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				err = checkDataPersistForMultiPvcs(f)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteMultiPvcsAndTestPodWithMultiPvcs()
				err = waitForTestPodGone(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
				for _, pvcName := range []string{"spdkcsi-pvc1", "spdkcsi-pvc2", "spdkcsi-pvc3"} {
					err = waitForPvcGone(f.ClientSet, pvcName)
					if err != nil {
						ginkgo.Fail(err.Error())
					}
				}
			})

			ginkgo.By("log verification for SMA workflow", func() {
				expLogerrMsgMap := map[string]string{
					"SMA.CreateDevice": "failed to catch the log about the SMA.CreateDevice phase",
					"SMA.AttachVolume": "failed to catch the log about the SMA.AttachVolume phase",
					"SMA.DetachVolume": "failed to catch the log about the SMA.DetachVolume phase",
				}
				err := verifyNodeServerLog(expLogerrMsgMap)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})
		})
	})
})
