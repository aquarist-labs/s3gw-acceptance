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

var _ = Describe("charts installation", func() {
	var suiteProperties map[string]interface{}
	chartName := "s3gw"

	BeforeEach(func() {
		suitePropertiesF, err := os.Open("suiteProperties.json")
		Expect(err).ToNot(HaveOccurred())
		defer suitePropertiesF.Close()
		byteValue, _ := io.ReadAll(suitePropertiesF)
		err = json.Unmarshal([]byte(byteValue), &suiteProperties)
		Expect(err).ToNot(HaveOccurred())
		Expect(suiteProperties).ToNot(BeNil())
	})

	AfterEach(func() {
	})

	Describe("s3gw-acceptance-0/s3gw-0", func() {
		imageName := "quay.io/s3gw/s3gw"
		namespace := "s3gw-acceptance-0"
		deploymentName := "s3gw-0"

		It("has the expected s3gw-0 deployment static values", func() {
			out, err := proc.Kubectl("get", "deployments",
				"-n", namespace,
				deploymentName,
				"-ojson")
			Expect(err).ToNot(HaveOccurred())

			var dJson map[string]interface{}
			err = json.Unmarshal([]byte(out), &dJson)
			Expect(err).ToNot(HaveOccurred())
			Expect(dJson).ToNot(BeNil())

			//deployment metadata
			Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(deploymentName))
			Expect(dJson["metadata"].(map[string]interface{})["namespace"].(string)).To(Equal(namespace))

			//annotations
			annotationsNode := dJson["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
			Expect(annotationsNode["deployment.kubernetes.io/revision"].(string)).To(Equal("1"))
			Expect(annotationsNode["meta.helm.sh/release-name"].(string)).To(Equal(deploymentName))
			Expect(annotationsNode["meta.helm.sh/release-namespace"].(string)).To(Equal(namespace))

			//labels
			labelNode := dJson["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
			Expect(labelNode["app.kubernetes.io/instance"].(string)).To(Equal(deploymentName))
			Expect(labelNode["app.kubernetes.io/managed-by"].(string)).To(Equal("Helm"))
			Expect(labelNode["app.kubernetes.io/name"].(string)).To(Equal(chartName))
			Expect(labelNode["app.kubernetes.io/version"].(string)).To(Equal("latest"))
			Expect(labelNode["helm.sh/chart"].(string)).To(Equal(chartName + "-" + suiteProperties["chartVersion"].(string)))

			//replicas
			Expect(dJson["spec"].(map[string]interface{})["replicas"].(float64)).To(BeEquivalentTo(1))

			//matching labels
			matchingLabelsNode := dJson["spec"].(map[string]interface{})["selector"].(map[string]interface{})["matchLabels"].(map[string]interface{})
			Expect(matchingLabelsNode["app.kubernetes.io/component"]).To(Equal("gateway"))
			Expect(matchingLabelsNode["app.kubernetes.io/instance"]).To(Equal(deploymentName))
			Expect(matchingLabelsNode["app.kubernetes.io/name"]).To(Equal(chartName))

			//strategy
			Expect(dJson["spec"].(map[string]interface{})["strategy"].(map[string]interface{})["type"].(string)).To(Equal("Recreate"))

			//spec template metadata labels
			specTemplateMetadataLables := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
			Expect(specTemplateMetadataLables["app.kubernetes.io/component"]).To(Equal("gateway"))
			Expect(specTemplateMetadataLables["app.kubernetes.io/instance"]).To(Equal(deploymentName))
			Expect(specTemplateMetadataLables["app.kubernetes.io/name"]).To(Equal(chartName))

			//containers
			containersNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})

			radosgwNode := containersNode[0].(map[string]interface{})
			radosgwArgsNode := radosgwNode["args"].([]interface{})

			//radosgw args
			Expect(radosgwArgsNode[0].(string)).To(Equal("--rgw-dns-name"))

			pubDNSName := deploymentName + "-" + namespace + "." + suiteProperties["S3GW_CLUSTER_IP"].(string) + "." + suiteProperties["S3GW_SYSTEM_DOMAIN"].(string)
			Expect(strings.Split(radosgwArgsNode[1].(string), ", ")[0]).To(BeEquivalentTo(pubDNSName))

			privDNSName := deploymentName + "-" + namespace + "." + namespace + ".svc.cluster.local"
			Expect(strings.Split(radosgwArgsNode[1].(string), ", ")[1]).To(BeEquivalentTo(privDNSName))

			Expect(radosgwArgsNode[2].(string)).To(Equal("--rgw-backend-store"))
			Expect(radosgwArgsNode[3].(string)).To(Equal("sfs"))
			Expect(radosgwArgsNode[4].(string)).To(Equal("--debug-rgw"))
			Expect(radosgwArgsNode[5].(string)).To(Equal("1"))
			Expect(radosgwArgsNode[6].(string)).To(Equal("--rgw_frontends"))
			Expect(radosgwArgsNode[7].(string)).To(Equal("beast port=7480 ssl_port=7481 ssl_certificate=/s3gw-cluster-ip-tls/tls.crt ssl_private_key=/s3gw-cluster-ip-tls/tls.key"))

			//envFrom
			radosgwEnvFromNode := radosgwNode["envFrom"].([]interface{})
			Expect(radosgwEnvFromNode[0].(map[string]interface{})["secretRef"].(map[string]interface{})["name"].(string)).To(Equal(deploymentName + "-" + namespace + "-creds"))

			//image
			Expect(radosgwNode["image"].(string)).To(Equal(imageName + ":" + suiteProperties["imageTag"].(string)))

			//imagePullPolicy
			Expect(radosgwNode["imagePullPolicy"].(string)).To(Equal("IfNotPresent"))

			//name
			Expect(radosgwNode["name"].(string)).To(Equal(deploymentName))

			radosgwPortsNode := radosgwNode["ports"].([]interface{})

			//ports
			Expect(radosgwPortsNode[0].(map[string]interface{})["containerPort"].(float64)).To(BeEquivalentTo(7480))
			Expect(radosgwPortsNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3"))
			Expect(radosgwPortsNode[0].(map[string]interface{})["protocol"].(string)).To(Equal("TCP"))

			Expect(radosgwPortsNode[1].(map[string]interface{})["containerPort"].(float64)).To(BeEquivalentTo(7481))
			Expect(radosgwPortsNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3-tls"))
			Expect(radosgwPortsNode[1].(map[string]interface{})["protocol"].(string)).To(Equal("TCP"))

			volumeMountsNode := radosgwNode["volumeMounts"].([]interface{})

			//volume mounts
			Expect(volumeMountsNode[0].(map[string]interface{})["mountPath"].(string)).To(Equal("/data"))
			Expect(volumeMountsNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3gw-lh-store"))

			Expect(volumeMountsNode[1].(map[string]interface{})["mountPath"].(string)).To(Equal("/s3gw-cluster-ip-tls"))
			Expect(volumeMountsNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3gw-cluster-ip-tls"))

			//volumes
			volumesNode := dJson["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"].([]interface{})

			Expect(volumesNode[0].(map[string]interface{})["name"].(string)).To(Equal("s3gw-lh-store"))
			Expect(volumesNode[0].(map[string]interface{})["persistentVolumeClaim"].(map[string]interface{})["claimName"]).To(Equal(deploymentName + "-pvc"))

			Expect(volumesNode[1].(map[string]interface{})["name"].(string)).To(Equal("s3gw-cluster-ip-tls"))
			Expect(volumesNode[1].(map[string]interface{})["secret"].(map[string]interface{})["secretName"]).To(Equal(deploymentName + "-" + namespace + "-cluster-ip-tls"))
		})
	})
})
