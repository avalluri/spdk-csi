package e2e

import (
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	smaNVMFConfigMapData = `{
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

var _ = ginkgo.Describe("SPDKCSI-SMA-NVMF", func() {
	f := framework.NewDefaultFramework("spdkcsi")
	ginkgo.BeforeEach(func() {
		deployConfigs(smaNVMFConfigMapData)
		deploySmaNvmfConfig()
		deployCsi()
	})

	ginkgo.AfterEach(func() {
		deleteCsi()
		deleteSmaNvmfConfig()
		deleteConfigs()
	})

	ginkgo.Context("Test SPDK CSI SMA NVMF", func() {
		ginkgo.FIt("Test SPDK CSI SMA NVMF", func() {
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
					"connected to SMA server 127.0.0.1:5114 with TargetType as xpu-sma-nvmftcp": "failed to catch the log about the connection to SMA server from node",
				}
				err := verifyNodeServerLog(expLogerrMsgMap)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			ginkgo.By("create a pvc and bind it to a pod", func() {
				deployPVC()
				deployTestPod()
				err := waitForTestPodReady(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteTestPod()
				err = waitForTestPodGone(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deletePVC()
				err = waitForPVCGone(f.ClientSet, "spdkcsi-pvc")
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			/*ginkgo.XIt("check data persistency after the pod is removed and recreated", func() {
				deployPVC()
				deployTestPod()
				err := waitForTestPodReady(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				err = checkDataPersist(f)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteTestPod()
				err = waitForTestPodGone(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deletePVC()
				err = waitForPVCGone(f.ClientSet, "spdkcsi-pvc")
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})

			ginkgo.XIt("create multiple pvcs and bind them to a pod", func() {
				deployMultiPVCs()
				deployTestPodWithMultiPVCs()
				err := waitForTestPodReady(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteTestPodWithMultiPVCs()
				err = waitForTestPodGone(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteMultiPVCs()
				for _, pvcName := range []string{"spdkcsi-pvc1", "spdkcsi-pvc2", "spdkcsi-pvc3"} {
					err = waitForPVCGone(f.ClientSet, pvcName)
					if err != nil {
						ginkgo.Fail(err.Error())
					}
				}
			})

			ginkgo.XIt("create multiple pvcs and a pod with multiple pvcs attached, and check data persistence after the pod is removed and recreated", func() {
				deployMultiPVCs()
				deployTestPodWithMultiPVCs()
				err := waitForTestPodReady(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				err = checkDataPersistForMultiPVCs(f)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteTestPodWithMultiPVCs()
				err = waitForTestPodGone(f.ClientSet)
				if err != nil {
					ginkgo.Fail(err.Error())
				}

				deleteMultiPVCs()
				for _, pvcName := range []string{"spdkcsi-pvc1", "spdkcsi-pvc2", "spdkcsi-pvc3"} {
					err = waitForPVCGone(f.ClientSet, pvcName)
					if err != nil {
						ginkgo.Fail(err.Error())
					}
				}
			})*/

			ginkgo.By("log verification for SMA workflow", func() {
				expLogerrMsgMap := map[string]string{
					"SMA.CreateDevice": "failed to catch the log about the SMA.CreateDevice phase",
					"SMA.AttachVolume": "failed to catch the log about the SMA.AttachVolume phase",
					"SMA.DetachVolume": "failed to catch the log about the SMA.DetachVolume phase",
					"SMA.DeleteDevice": "failed to catch the log about the SMA.DeleteDevice phase",
				}
				err := verifyNodeServerLog(expLogerrMsgMap)
				if err != nil {
					ginkgo.Fail(err.Error())
				}
			})
		})
	})
})
