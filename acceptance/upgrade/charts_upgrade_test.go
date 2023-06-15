// Copyright © 2023 SUSE LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package upgrade_test

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	. "github.com/aquarist-labs/s3gw/acceptance/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("charts upgrades", Label("Charts"), func() {
	var suiteProperties map[string]interface{}
	chartsRoot := "s3gw/s3gw"
	chartName := "s3gw"
	s3gwImageName := "quay.io/s3gw/s3gw"
	s3gwUiImageName := "quay.io/s3gw/s3gw-ui"

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

	Context("Upgrading s3gw chart [previous -> target], default installation", Label("Default"), func() {
		namespace := NanoSecName("s3gw")
		releaseName := NanoSecName("s3gw")
		expectedRevisionOnUpgrade := "2"

		BeforeEach(func() {
			if len(suiteProperties["RELEASE"].(string)) > 0 {
				releaseName = suiteProperties["RELEASE"].(string)
			}
			if len(suiteProperties["NAMESPACE"].(string)) > 0 {
				namespace = suiteProperties["NAMESPACE"].(string)
			}
			if len(suiteProperties["EXPECTED_REVISION_ON_UPGRADE"].(string)) > 0 {
				expectedRevisionOnUpgrade = suiteProperties["EXPECTED_REVISION_ON_UPGRADE"].(string)
			}

			argsPrev := []string{"install", "--create-namespace", "-n", namespace,
				releaseName, chartsRoot,
				"--version", suiteProperties["CHARTS_VER_PREV"].(string),
				"--wait",
				"--set", "publicDomain=" + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "ui.publicDomain=" + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string)}

			if extraArgsPrev := suiteProperties["CHARTS_PREV_EXTRA_ARGS"].(string); len(extraArgsPrev) > 0 {
				out, err := Run("../..", true, "helm", append(argsPrev, strings.Split(extraArgsPrev, " ")...)...)
				Expect(err).ToNot(HaveOccurred(), out)
			} else {
				out, err := Run("../..", true, "helm", argsPrev...)
				Expect(err).ToNot(HaveOccurred(), out)
			}

			argsCurr := []string{"upgrade", releaseName, "-n", namespace, chartsRoot,
				"--version", suiteProperties["CHARTS_VER"].(string),
				"--wait",
				"--set", "publicDomain=" + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "ui.publicDomain=" + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string),
				"--set", "storageClass.name=local-path"}

			if extraArgsCurr := suiteProperties["CHARTS_EXTRA_ARGS"].(string); len(extraArgsCurr) > 0 {
				out, err := Run("../..", true, "helm", append(argsCurr, strings.Split(extraArgsCurr, " ")...)...)
				Expect(err).ToNot(HaveOccurred(), out)
			} else {
				out, err := Run("../..", true, "helm", argsCurr...)
				Expect(err).ToNot(HaveOccurred(), out)
			}
		})

		AfterEach(func() {
			out, err := Run("../..", true, "helm", "uninstall", "-n", namespace, releaseName, "--wait")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		It("deployed resources have [target] version properties", func() {
			By("getting the s3gw deployment", func() {
				out, err := Kubectl("get", "deployments",
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
				Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal(expectedRevisionOnUpgrade))
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
				out, err := Kubectl("get", "deployments",
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
				Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal(expectedRevisionOnUpgrade))
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
})
