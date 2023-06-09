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

package cosi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	. "github.com/aquarist-labs/s3gw/acceptance/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const bucketClassFMT = `
kind: BucketClass
apiVersion: objectstorage.k8s.io/v1alpha1
metadata:
    name: %s
driverName: %s
deletionPolicy: %s
`

const bucketAccessClassFMT = `
kind: BucketAccessClass
apiVersion: objectstorage.k8s.io/v1alpha1
metadata:
    name: %s
driverName: %s
authenticationType: %s
`

const bucketClaimFMT = `
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketClaim
metadata:
    namespace: %s
    name: %s
spec:
    bucketClassName: %s
    protocols:
        - s3
`

var _ = Describe("COSI workflow - single instance", Label("COSI"), func() {
	var suiteProperties map[string]interface{}
	chartsRoot := "charts/charts/s3gw"
	namespace := NanoSecName("s3gw-cosi-workflow")
	releaseName := NanoSecName("s3gw-cosi-workflow")
	driverName := releaseName + "." + namespace + ".objectstorage.k8s.io"

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
		out, err := Run("../..", true, "helm", "install", "--create-namespace", "-n", namespace,
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
		out, err := Run("../..", true, "helm", "uninstall", "-n", namespace, releaseName, "--wait")
		Expect(err).ToNot(HaveOccurred(), out)
	})

	When("specifying deletionPolicy:Delete in BucketClass", func() {
		bucketClassName := "bucket-class-delete"
		deletionPolicy := "Delete"

		BeforeEach(func() {
			BucketClassFileName := NanoSecName("BucketClass") + ".yaml"
			if BucketClassFile, err := os.Create(BucketClassFileName); err != nil {
				//make this fail
				Expect(err).ToNot(HaveOccurred())
			} else {
				defer os.Remove(BucketClassFileName)
				if _, err := fmt.Fprintf(BucketClassFile, bucketClassFMT,
					bucketClassName,
					driverName,
					deletionPolicy); err != nil {
					//make this fail
					Expect(err).ToNot(HaveOccurred())
				} else {
					out, err := Kubectl("apply", "-f", BucketClassFileName)
					Expect(err).ToNot(HaveOccurred(), out)

					By("checking BucketClass", func() {
						out, err = Kubectl("get", "bucketclass", bucketClassName, "-ojson")
						Expect(err).ToNot(HaveOccurred(), out)

						var dJson map[string]interface{}
						err = json.Unmarshal([]byte(out), &dJson)
						Expect(err).ToNot(HaveOccurred())
						Expect(dJson).ToNot(BeNil())

						Expect(dJson["deletionPolicy"].(string)).To(Equal(deletionPolicy))
						Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(bucketClassName))
					})
				}
			}
		})

		AfterEach(func() {
			out, err := Kubectl("delete", "bucketclass", bucketClassName)
			Expect(err).ToNot(HaveOccurred(), out)
		})

		When("specifying authenticationType:KEY in BucketAccessClass", func() {
			bucketAccessClassName := "bucket-access-class-key"
			authenticationType := "KEY"

			BeforeEach(func() {
				BucketAccessClassFileName := NanoSecName("BucketAccessClass") + ".yaml"
				if BucketAccessClassFile, err := os.Create(BucketAccessClassFileName); err != nil {
					//make this fail
					Expect(err).ToNot(HaveOccurred())
				} else {
					defer os.Remove(BucketAccessClassFileName)
					if _, err := fmt.Fprintf(BucketAccessClassFile, bucketAccessClassFMT,
						bucketAccessClassName,
						driverName,
						authenticationType); err != nil {
						//make this fail
						Expect(err).ToNot(HaveOccurred())
					} else {
						out, err := Kubectl("apply", "-f", BucketAccessClassFileName)
						Expect(err).ToNot(HaveOccurred(), out)

						By("checking BucketAccessClass", func() {
							out, err = Kubectl("get", "bucketaccessclass", bucketAccessClassName, "-ojson")
							Expect(err).ToNot(HaveOccurred(), out)

							var dJson map[string]interface{}
							err = json.Unmarshal([]byte(out), &dJson)
							Expect(err).ToNot(HaveOccurred())
							Expect(dJson).ToNot(BeNil())

							Expect(dJson["authenticationType"].(string)).To(Equal(authenticationType))
							Expect(dJson["metadata"].(map[string]interface{})["name"].(string)).To(Equal(bucketAccessClassName))
						})
					}
				}
			})

			AfterEach(func() {
				out, err := Kubectl("delete", "bucketaccessclass", bucketAccessClassName)
				Expect(err).ToNot(HaveOccurred(), out)
			})

			When("creating a BucketClaim", func() {
				bucketClaimName := "bucket-claim-0"

				BeforeEach(func() {
					BucketClaimFileName := NanoSecName("BucketClaim") + ".yaml"
					if BucketClaimFile, err := os.Create(BucketClaimFileName); err != nil {
						//make this fail
						Expect(err).ToNot(HaveOccurred())
					} else {
						defer os.Remove(BucketClaimFileName)
						if _, err := fmt.Fprintf(BucketClaimFile, bucketClaimFMT,
							namespace,
							bucketClaimName,
							bucketClassName); err != nil {
							//make this fail
							Expect(err).ToNot(HaveOccurred())
						} else {
							out, err := Kubectl("apply", "-f", BucketClaimFileName)
							Expect(err).ToNot(HaveOccurred(), out)

							Eventually(func() bool {
								out, err = Kubectl("get", "bucketclaim", "-n", namespace, bucketClaimName, "-ojson")
								Expect(err).ToNot(HaveOccurred(), out)

								var dJson map[string]interface{}
								err = json.Unmarshal([]byte(out), &dJson)
								Expect(err).ToNot(HaveOccurred())
								Expect(dJson).ToNot(BeNil())

								return dJson["status"].(map[string]interface{})["bucketReady"].(bool)
							}, "1m").Should(Equal(true))
						}
					}
				})

				AfterEach(func() {
					out, err := Kubectl("delete", "bucketclaim", "-n", namespace, bucketClaimName)
					Expect(err).ToNot(HaveOccurred(), out)
				})

				It("deploys expected resources", func() {
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

					})
				})
			})
		})
	})
})
