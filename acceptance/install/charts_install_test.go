// Copyright © 2021 - 2023 SUSE LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package install_test

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/aquarist-labs/s3gw/acceptance/helpers/proc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("charts installations", Label("Charts"), func() {
	var suiteProperties map[string]interface{}
	chartsRoot := "charts/charts/s3gw"
	chartName := "s3gw"
	s3gwImageName := "quay.io/s3gw/s3gw"
	s3gwUiImageName := "quay.io/s3gw/s3gw-ui"
	s3gwCOSIDriverImageName := "quay.io/s3gw/s3gw-cosi-driver"
	s3gwCOSISidecarImageName := "quay.io/s3gw/s3gw-cosi-sidecar"

	BeforeEach(func() {
		if suitePropertiesF, err := os.Open("../suiteProperties.json"); err == nil {
			defer suitePropertiesF.Close()
			byteValue, _ := io.ReadAll(suitePropertiesF)
			err = json.Unmarshal([]byte(byteValue), &suiteProperties)
			Expect(err).ToNot(HaveOccurred())
		} else {
			//make this fail
			Expect(err).ToNot(HaveOccurred())
			Expect(suiteProperties).ToNot(BeNil())
		}
	})

	AfterEach(func() {
	})

	Context("deploying s3gw-def/s3gw-def", Label("Default"), func() {
		namespace := "s3gw-def"
		releaseName := "s3gw-def"

		BeforeEach(func() {
			out, err := proc.Run("../..", true, "helm", "install", "--create-namespace", "-n", namespace,
				"--set", "publicDomain="+suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "ui.publicDomain="+suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "ui.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "cosi.driver.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "cosi.sidecar.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				releaseName, chartsRoot, "--wait")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		AfterEach(func() {
			out, err := proc.Run("../..", true, "helm", "uninstall", "-n", namespace, releaseName, "--wait")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		It("deploys expected resources", func() {
			By("getting the s3gw deployment", func() {
				out, err := proc.Kubectl("get", "deployments",
					"-n", namespace,
					releaseName,
					"-ojson")
				Expect(err).ToNot(HaveOccurred())

				var dJson map[string]interface{}
				err = json.Unmarshal([]byte(out), &dJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(dJson).ToNot(BeNil())

				//deployment metadata
				Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(releaseName))
				Expect(dJson["metadata"].(map[string]interface{})["namespace"].(string)).To(Equal(namespace))

				//annotations
				annotationsNode := dJson["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
				Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal("1"))
				Expect(annotationsNode["meta.helm.sh/release-name"].(string)).To(Equal(releaseName))
				Expect(annotationsNode["meta.helm.sh/release-namespace"].(string)).To(Equal(namespace))

				//labels
				labelNode := dJson["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(labelNode["app.kubernetes.io/instance"].(string)).To(Equal(releaseName))
				Expect(labelNode["app.kubernetes.io/managed-by"].(string)).To(Equal("Helm"))
				Expect(labelNode["app.kubernetes.io/name"].(string)).To(Equal(chartName))
				Expect(labelNode["app.kubernetes.io/version"].(string)).To(Equal("latest"))
				Expect(labelNode["helm.sh/chart"].(string)).To(Equal(chartName + "-" + suiteProperties["CHARTS_VER"].(string)))

				//replicas
				Expect(dJson["spec"].(map[string]interface{})["replicas"].(float64)).To(BeEquivalentTo(1))

				//matching labels
				matchingLabelsNode := dJson["spec"].(map[string]interface{})["selector"].(map[string]interface{})["matchLabels"].(map[string]interface{})
				Expect(matchingLabelsNode["app.kubernetes.io/component"]).To(Equal("gateway"))
				Expect(matchingLabelsNode["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(matchingLabelsNode["app.kubernetes.io/name"]).To(Equal(chartName))

				//strategy
				Expect(dJson["spec"].(map[string]interface{})["strategy"].(map[string]interface{})["type"].(string)).To(Equal("Recreate"))

				//spec template metadata labels
				specTemplateMetadataLables := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(specTemplateMetadataLables["app.kubernetes.io/component"]).To(Equal("gateway"))
				Expect(specTemplateMetadataLables["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(specTemplateMetadataLables["app.kubernetes.io/name"]).To(Equal(chartName))

				//containers
				containersNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})

				cnt0Node := containersNode[0].(map[string]interface{})
				cnt0ArgsNode := cnt0Node["args"].([]interface{})

				//radosgw args
				Expect(cnt0ArgsNode[0].(string)).To(Equal("--rgw-dns-name"))

				pubDNSName := releaseName + "-" + namespace + "." + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string)
				Expect(strings.Split(cnt0ArgsNode[1].(string), ", ")[0]).To(BeEquivalentTo(pubDNSName))

				privDNSName := releaseName + "-" + namespace + "." + namespace + ".svc.cluster.local"
				Expect(strings.Split(cnt0ArgsNode[1].(string), ", ")[1]).To(BeEquivalentTo(privDNSName))

				Expect(cnt0ArgsNode[2].(string)).To(Equal("--rgw-backend-store"))
				Expect(cnt0ArgsNode[3].(string)).To(Equal("sfs"))
				Expect(cnt0ArgsNode[4].(string)).To(Equal("--debug-rgw"))
				Expect(cnt0ArgsNode[5].(string)).To(Equal("1"))
				Expect(cnt0ArgsNode[6].(string)).To(Equal("--rgw_frontends"))
				Expect(cnt0ArgsNode[7].(string)).To(Equal("beast port=7480 ssl_port=7481 ssl_certificate=/s3gw-cluster-ip-tls/tls.crt ssl_private_key=/s3gw-cluster-ip-tls/tls.key"))

				//envFrom
				cnt0EnvFromNode := cnt0Node["envFrom"].([]interface{})
				Expect(cnt0EnvFromNode[0].(map[string]interface{})["secretRef"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-" + namespace + "-creds"))

				//image
				Expect(cnt0Node["image"].(string)).To(Equal(s3gwImageName + ":v" + suiteProperties["IMAGE_TAG"].(string)))

				//imagePullPolicy
				Expect(cnt0Node["imagePullPolicy"].(string)).To(Equal("IfNotPresent"))

				//name
				Expect(cnt0Node["name"].(string)).To(Equal(releaseName))

				cnt0PortsNode := cnt0Node["ports"].([]interface{})

				//ports
				Expect(cnt0PortsNode[0].(map[string]interface{})["containerPort"].(float64)).To(BeEquivalentTo(7480))
				Expect(cnt0PortsNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3"))
				Expect(cnt0PortsNode[0].(map[string]interface{})["protocol"].(string)).To(Equal("TCP"))

				Expect(cnt0PortsNode[1].(map[string]interface{})["containerPort"].(float64)).To(BeEquivalentTo(7481))
				Expect(cnt0PortsNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3-tls"))
				Expect(cnt0PortsNode[1].(map[string]interface{})["protocol"].(string)).To(Equal("TCP"))

				volumeMountsNode := cnt0Node["volumeMounts"].([]interface{})

				//volume mounts
				Expect(volumeMountsNode[0].(map[string]interface{})["mountPath"].(string)).To(Equal("/data"))
				Expect(volumeMountsNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3gw-lh-store"))

				Expect(volumeMountsNode[1].(map[string]interface{})["mountPath"].(string)).To(Equal("/s3gw-cluster-ip-tls"))
				Expect(volumeMountsNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3gw-cluster-ip-tls"))

				//volumes
				volumesNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"].([]interface{})

				Expect(volumesNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3gw-lh-store"))
				Expect(volumesNode[0].(map[string]interface{})["persistentVolumeClaim"].(map[string]interface{})["claimName"]).To(Equal(releaseName + "-pvc"))

				Expect(volumesNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3gw-cluster-ip-tls"))
				Expect(volumesNode[1].(map[string]interface{})["secret"].(map[string]interface{})["secretName"]).To(Equal(releaseName + "-" + namespace + "-cluster-ip-tls"))
			})

			By("getting the s3gw-ui deployment", func() {
				out, err := proc.Kubectl("get", "deployments",
					"-n", namespace,
					releaseName+"-ui",
					"-ojson")
				Expect(err).ToNot(HaveOccurred())

				var dJson map[string]interface{}
				err = json.Unmarshal([]byte(out), &dJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(dJson).ToNot(BeNil())

				//deployment metadata
				Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-ui"))
				Expect(dJson["metadata"].(map[string]interface{})["namespace"].(string)).To(Equal(namespace))

				//annotations
				annotationsNode := dJson["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
				Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal("1"))
				Expect(annotationsNode["meta.helm.sh/release-name"].(string)).To(Equal(releaseName))
				Expect(annotationsNode["meta.helm.sh/release-namespace"].(string)).To(Equal(namespace))

				//labels
				labelNode := dJson["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(labelNode["app.kubernetes.io/instance"].(string)).To(Equal(releaseName))
				Expect(labelNode["app.kubernetes.io/managed-by"].(string)).To(Equal("Helm"))
				Expect(labelNode["app.kubernetes.io/name"].(string)).To(Equal(chartName))
				Expect(labelNode["app.kubernetes.io/version"].(string)).To(Equal("latest"))
				Expect(labelNode["helm.sh/chart"].(string)).To(Equal(chartName + "-" + suiteProperties["CHARTS_VER"].(string)))

				//replicas
				Expect(dJson["spec"].(map[string]interface{})["replicas"].(float64)).To(BeEquivalentTo(1))

				//matching labels
				matchingLabelsNode := dJson["spec"].(map[string]interface{})["selector"].(map[string]interface{})["matchLabels"].(map[string]interface{})
				Expect(matchingLabelsNode["app.kubernetes.io/component"]).To(Equal("ui"))
				Expect(matchingLabelsNode["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(matchingLabelsNode["app.kubernetes.io/name"]).To(Equal(chartName))

				//strategy
				Expect(dJson["spec"].(map[string]interface{})["strategy"].(map[string]interface{})["type"].(string)).To(Equal("RollingUpdate"))

				//spec template metadata labels
				specTemplateMetadataLables := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(specTemplateMetadataLables["app.kubernetes.io/component"]).To(Equal("ui"))
				Expect(specTemplateMetadataLables["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(specTemplateMetadataLables["app.kubernetes.io/name"]).To(Equal(chartName))

				//containers
				containersNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})
				cnt0Node := containersNode[0].(map[string]interface{})

				//envFrom
				cnt0EnvFromNode := cnt0Node["envFrom"].([]interface{})
				Expect(cnt0EnvFromNode[0].(map[string]interface{})["configMapRef"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-" + namespace + "-config"))
				Expect(cnt0EnvFromNode[1].(map[string]interface{})["secretRef"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-" + namespace + "-creds"))

				//image
				Expect(cnt0Node["image"].(string)).To(Equal(s3gwUiImageName + ":v" + suiteProperties["IMAGE_TAG"].(string)))

				//imagePullPolicy
				Expect(cnt0Node["imagePullPolicy"].(string)).To(Equal("IfNotPresent"))

				//name
				Expect(cnt0Node["name"].(string)).To(Equal("s3gw-ui"))

				cnt0PortsNode := cnt0Node["ports"].([]interface{})

				//ports
				Expect(cnt0PortsNode[0].(map[string]interface{})["containerPort"].(float64)).To(BeEquivalentTo(8080))
				Expect(cnt0PortsNode[0].(map[string]interface{})["protocol"].(string)).To(Equal("TCP"))
			})
		})
	})

	Context("deploying s3gw-acceptance-cosi/s3gw-cosi", Label("COSI"), func() {
		namespace := "s3gw-acceptance-cosi"
		releaseName := "s3gw-cosi"

		BeforeEach(func() {
			out, err := proc.Run("../..", true, "helm", "install", "--create-namespace", "-n", namespace,
				"--set", "publicDomain="+suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "ui.publicDomain="+suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "ui.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "cosi.driver.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "cosi.sidecar.imageTag=v"+suiteProperties["IMAGE_TAG"].(string),
				"--set", "cosi.enabled=true",
				releaseName, chartsRoot, "--wait")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		AfterEach(func() {
			out, err := proc.Run("../..", true, "helm", "uninstall", "-n", namespace, releaseName, "--wait")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		It("has the expected s3gw-cosi deployment static values", func() {
			By("getting the objectstorage-provisioner deployment", func() {
				out, err := proc.Kubectl("get", "deployments",
					"-n", namespace,
					releaseName+"-objectstorage-provisioner",
					"-ojson")
				Expect(err).ToNot(HaveOccurred())

				var dJson map[string]interface{}
				err = json.Unmarshal([]byte(out), &dJson)
				Expect(err).ToNot(HaveOccurred())
				Expect(dJson).ToNot(BeNil())

				//deployment metadata
				Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-objectstorage-provisioner"))
				Expect(dJson["metadata"].(map[string]interface{})["namespace"].(string)).To(Equal(namespace))

				//annotations
				annotationsNode := dJson["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
				Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal("1"))
				Expect(annotationsNode["meta.helm.sh/release-name"].(string)).To(Equal(releaseName))
				Expect(annotationsNode["meta.helm.sh/release-namespace"].(string)).To(Equal(namespace))

				//labels
				labelNode := dJson["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(labelNode["app.kubernetes.io/instance"].(string)).To(Equal(releaseName))
				Expect(labelNode["app.kubernetes.io/managed-by"].(string)).To(Equal("Helm"))
				Expect(labelNode["app.kubernetes.io/name"].(string)).To(Equal(chartName))
				Expect(labelNode["app.kubernetes.io/version"].(string)).To(Equal("latest"))
				Expect(labelNode["helm.sh/chart"].(string)).To(Equal(chartName + "-" + suiteProperties["CHARTS_VER"].(string)))

				//replicas
				Expect(dJson["spec"].(map[string]interface{})["replicas"].(float64)).To(BeEquivalentTo(1))

				//matching labels
				matchingLabelsNode := dJson["spec"].(map[string]interface{})["selector"].(map[string]interface{})["matchLabels"].(map[string]interface{})
				Expect(matchingLabelsNode["app.kubernetes.io/component"]).To(Equal("cosi"))
				Expect(matchingLabelsNode["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(matchingLabelsNode["app.kubernetes.io/name"]).To(Equal(chartName))

				//strategy
				Expect(dJson["spec"].(map[string]interface{})["strategy"].(map[string]interface{})["type"].(string)).To(Equal("Recreate"))

				//spec template metadata labels
				specTemplateMetadataLables := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
				Expect(specTemplateMetadataLables["app.kubernetes.io/component"]).To(Equal("cosi"))
				Expect(specTemplateMetadataLables["app.kubernetes.io/instance"]).To(Equal(releaseName))
				Expect(specTemplateMetadataLables["app.kubernetes.io/name"]).To(Equal(chartName))

				//containers
				containersNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})

				cnt0Node := containersNode[0].(map[string]interface{})

				//envFrom
				cnt0EnvFromNode := cnt0Node["envFrom"].([]interface{})
				Expect(cnt0EnvFromNode[0].(map[string]interface{})["secretRef"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-" + namespace + "-objectstorage-provisioner"))

				//image
				Expect(cnt0Node["image"].(string)).To(Equal(s3gwCOSIDriverImageName + ":v" + suiteProperties["IMAGE_TAG"].(string)))

				//imagePullPolicy
				Expect(cnt0Node["imagePullPolicy"].(string)).To(Equal("IfNotPresent"))

				//name
				Expect(cnt0Node["name"].(string)).To(Equal(releaseName + "-cosi-driver"))

				volumeMountsNode := cnt0Node["volumeMounts"].([]interface{})

				//volume mounts
				Expect(volumeMountsNode[0].(map[string]interface{})["mountPath"].(string)).To(Equal("/var/lib/cosi"))
				Expect(volumeMountsNode[0].(map[string]interface{})["name"].(string)).To(Equal("socket"))

				cnt1Node := containersNode[1].(map[string]interface{})

				//envFrom
				cnt1EnvFromNode := cnt1Node["envFrom"].([]interface{})
				Expect(cnt1EnvFromNode[0].(map[string]interface{})["secretRef"].(map[string]interface{})["name"].(string)).To(Equal(releaseName + "-" + namespace + "-objectstorage-provisioner"))

				//image
				Expect(cnt1Node["image"].(string)).To(Equal(s3gwCOSISidecarImageName + ":v" + suiteProperties["IMAGE_TAG"].(string)))

				cnt1ArgsNode := cnt1Node["args"].([]interface{})

				//cnt1 args
				Expect(cnt1ArgsNode[0].(string)).To(Equal("--v=5"))

				cnt1EnvNode := cnt1Node["env"].([]interface{})

				//name
				Expect(cnt1EnvNode[0].(map[string]interface{})["name"].(string)).To(Equal("POD_NAMESPACE"))

				volumeMountsNode = cnt1Node["volumeMounts"].([]interface{})

				//volume mounts
				Expect(volumeMountsNode[0].(map[string]interface{})["mountPath"].(string)).To(Equal("/var/lib/cosi"))
				Expect(volumeMountsNode[0].(map[string]interface{})["name"].(string)).To(Equal("socket"))

				//serviceAccount
				serviceAccount := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["serviceAccount"].(string)
				Expect(serviceAccount).To(Equal(releaseName + "-" + namespace + "-objectstorage-provisioner-sa"))

				//serviceAccountName
				serviceAccountName := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["serviceAccountName"].(string)
				Expect(serviceAccountName).To(Equal(releaseName + "-" + namespace + "-objectstorage-provisioner-sa"))

				//volumes
				volumesNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"].([]interface{})

				Expect(volumesNode[0].(map[string]interface{})["name"].(string)).To(Equal("socket"))
			})
		})
	})
})
