package test

import (
	goctx "context"
	"fmt"

	asdbv1beta1 "github.com/aerospike/aerospike-kubernetes-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe(
	"AerospikeCluster", func() {

		ctx := goctx.TODO()

		// Cluster lifecycle related
		Context(
			"DeployClusterPost490", func() {
				DeployClusterForAllImagesPost490(ctx)
			},
		)
		Context(
			"DeployClusterDiffStorageMultiPodPerHost", func() {
				DeployClusterForDiffStorageTest(ctx, 2, true)
			},
		)
		Context(
			"DeployClusterDiffStorageSinglePodPerHost", func() {
				DeployClusterForDiffStorageTest(ctx, 2, false)
			},
		)
		Context(
			"DeployClusterWithDNSConfiguration", func() {
				DeployClusterWithDNSConfiguration(ctx)
			},
		)
		Context(
			"CommonNegativeClusterValidationTest", func() {
				NegativeClusterValidationTest(ctx)
			},
		)
		Context(
			"UpdateCluster", func() {
				UpdateClusterTest(ctx)
			},
		)
		Context(
			"ScaleDownWithMigrateFillDelay", func() {
				clusterNamespacedName := getClusterNamespacedName(
					"migrate-fill-delay-cluster", namespace)
				migrateFillDelay := int64(120)
				BeforeEach(
					func() {
						aeroCluster := createDummyAerospikeCluster(clusterNamespacedName, 4)
						aeroCluster.Spec.AerospikeConfig.Value["service"].(map[string]interface{})["migrate-fill-delay"] = migrateFillDelay
						err := deployCluster(k8sClient, ctx, aeroCluster)
						Expect(err).ToNot(HaveOccurred())
					},
				)

				AfterEach(
					func() {
						aeroCluster, err := getCluster(
							k8sClient, ctx, clusterNamespacedName,
						)
						Expect(err).ToNot(HaveOccurred())

						_ = deleteCluster(k8sClient, ctx, aeroCluster)
					},
				)

				It("Should ignore migrate-fill-delay while scaling down", func() {

					aeroCluster, err := getCluster(k8sClient, ctx, clusterNamespacedName)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.Size = aeroCluster.Spec.Size - 2
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())

					// verify that migrate-fill-delay is set to 0 while scaling down
					err = validateMigrateFillDelay(ctx, k8sClient, logger, clusterNamespacedName, 0)
					Expect(err).ToNot(HaveOccurred())

					err = waitForAerospikeCluster(
						k8sClient, ctx, aeroCluster, int(aeroCluster.Spec.Size), retryInterval,
						getTimeout(2),
					)
					Expect(err).ToNot(HaveOccurred())

					// verify that migrate-fill-delay is reverted to original value after scaling down
					err = validateMigrateFillDelay(ctx, k8sClient, logger, clusterNamespacedName, migrateFillDelay)
					Expect(err).ToNot(HaveOccurred())
				})
			})
	},
)

// Test cluster deployment with all image post 4.9.0
func DeployClusterForAllImagesPost490(ctx goctx.Context) {

	// post 4.9.0, need feature-key file
	versions := []string{
		"5.7.0.8", "5.6.0.7", "5.5.0.3", "5.4.0.5", "5.3.0.10", "5.2.0.17",
		"5.1.0.25", "5.0.0.21",
		"4.9.0.11",
	}

	for _, v := range versions {

		It(
			fmt.Sprintf("Deploy-%s", v), func() {
				clusterName := "deploy-cluster"
				clusterNamespacedName := getClusterNamespacedName(
					clusterName, namespace,
				)

				image := fmt.Sprintf(
					"aerospike/aerospike-server-enterprise:%s", v,
				)
				aeroCluster, err := getAeroClusterConfig(
					clusterNamespacedName, image,
				)
				Expect(err).ToNot(HaveOccurred())

				err = deployCluster(k8sClient, ctx, aeroCluster)
				Expect(err).ToNot(HaveOccurred())

				_ = deleteCluster(k8sClient, ctx, aeroCluster)
			},
		)
	}
}

// Test cluster deployment with different namespace storage
func DeployClusterForDiffStorageTest(
	ctx goctx.Context, nHosts int32, multiPodPerHost bool,
) {
	clusterSz := nHosts
	if multiPodPerHost {
		clusterSz++
	}
	repFact := nHosts

	Context(
		"Positive", func() {
			// Cluster with n nodes, enterprise can be more than 8
			// Cluster with resources
			// Verify: Connect with cluster

			// Namespace storage configs
			//
			// SSD Storage Engine
			It(
				"SSDStorageCluster", func() {
					clusterNamespacedName := getClusterNamespacedName(
						"ssdstoragecluster", namespace,
					)
					aeroCluster := createSSDStorageCluster(
						clusterNamespacedName, clusterSz, repFact,
						multiPodPerHost,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
					_ = deleteCluster(k8sClient, ctx, aeroCluster)
				},
			)

			// HDD Storage Engine with Data in Memory
			It(
				"HDDAndDataInMemStorageCluster", func() {
					clusterNamespacedName := getClusterNamespacedName(
						"inmemstoragecluster", namespace,
					)

					aeroCluster := createHDDAndDataInMemStorageCluster(
						clusterNamespacedName, clusterSz, repFact,
						multiPodPerHost,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
					err = deleteCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
				},
			)
			// HDD Storage Engine with Data in Index Engine
			It(
				"HDDAndDataInIndexStorageCluster", func() {
					clusterNamespacedName := getClusterNamespacedName(
						"datainindexcluster", namespace,
					)

					aeroCluster := createHDDAndDataInIndexStorageCluster(
						clusterNamespacedName, clusterSz, repFact,
						multiPodPerHost,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
					err = deleteCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
				},
			)
			// Data in Memory Without Persistence
			It(
				"DataInMemWithoutPersistentStorageCluster", func() {
					clusterNamespacedName := getClusterNamespacedName(
						"nopersistentcluster", namespace,
					)

					aeroCluster := createDataInMemWithoutPersistentStorageCluster(
						clusterNamespacedName, clusterSz, repFact,
						multiPodPerHost,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
					err = deleteCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
				},
			)
			// Shadow Device
			// It("ShadowDeviceStorageCluster", func() {
			// 	aeroCluster := createShadowDeviceStorageCluster(clusterNamespacedName, clusterSz, repFact, multiPodPerHost)
			// 	if err := deployCluster(k8sClient, ctx, aeroCluster); err != nil {
			// 		t.Fatal(err)
			// 	}
			// 	// make info call

			// 	deleteCluster(k8sClient, ctx, aeroCluster)
			// })

			// Persistent Memory (pmem) Storage Engine

		},
	)
}

func DeployClusterWithDNSConfiguration(ctx goctx.Context) {
	var aeroCluster *asdbv1beta1.AerospikeCluster

	It(
		"deploy with dnsPolicy 'None' and dnsConfig given",
		func() {
			clusterNamespacedName := getClusterNamespacedName(
				"dns-config-cluster", namespace,
			)
			aeroCluster = createDummyAerospikeCluster(clusterNamespacedName, 2)
			noneDNS := v1.DNSNone
			aeroCluster.Spec.PodSpec.InputDNSPolicy = &noneDNS

			var kubeDNSSvc v1.Service

			// fetch kube-dns service to get the DNS server IP for DNS lookup
			// This service name is same for both kube-dns and coreDNS DNS servers
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      "kube-dns",
			}, &kubeDNSSvc)).ShouldNot(HaveOccurred())

			dnsConfig := &v1.PodDNSConfig{
				Nameservers: kubeDNSSvc.Spec.ClusterIPs,
				Searches:    []string{"svc.cluster.local", "cluster.local"},
				Options: []v1.PodDNSConfigOption{
					{
						Name:  "ndots",
						Value: func(input string) *string { return &input }("5"),
					}},
			}
			aeroCluster.Spec.PodSpec.DNSConfig = dnsConfig

			err := deployCluster(k8sClient, ctx, aeroCluster)
			Expect(err).ShouldNot(HaveOccurred())

			sts, err := getSTSFromRackID(aeroCluster, 0)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(sts.Spec.Template.Spec.DNSConfig).To(Equal(dnsConfig))
		},
	)

	AfterEach(func() {
		_ = deleteCluster(k8sClient, ctx, aeroCluster)
	})
}

// Test cluster cr updation
func UpdateClusterTest(ctx goctx.Context) {

	clusterName := "update-cluster"
	clusterNamespacedName := getClusterNamespacedName(clusterName, namespace)

	// Note: this storage will be used by dynamically added namespace after deployment of cluster
	dynamicNsPath := "/test/dev/dynamicns"
	dynamicNsVolume := asdbv1beta1.VolumeSpec{
		Name: "dynamicns",
		Source: asdbv1beta1.VolumeSource{
			PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
				Size:         resource.MustParse("1Gi"),
				StorageClass: storageClass,
				VolumeMode:   v1.PersistentVolumeBlock,
			},
		},
		Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
			Path: dynamicNsPath,
		},
	}
	dynamicNsPath1 := "/test/dev/dynamicns1"
	dynamicNsVolume1 := asdbv1beta1.VolumeSpec{
		Name: "dynamicns1",
		Source: asdbv1beta1.VolumeSource{
			PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
				Size:         resource.MustParse("1Gi"),
				StorageClass: storageClass,
				VolumeMode:   v1.PersistentVolumeBlock,
			},
		},
		Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
			Path: dynamicNsPath1,
		},
	}
	dynamicNs := map[string]interface{}{
		"name":               "dynamicns",
		"memory-size":        1000955200,
		"replication-factor": 2,
		"storage-engine": map[string]interface{}{
			"type":    "device",
			"devices": []interface{}{dynamicNsPath, dynamicNsPath1},
		},
	}

	BeforeEach(
		func() {
			aeroCluster := createDummyAerospikeCluster(clusterNamespacedName, 3)
			aeroCluster.Spec.Storage.Volumes = append(
				aeroCluster.Spec.Storage.Volumes, dynamicNsVolume, dynamicNsVolume1,
			)

			err := deployCluster(k8sClient, ctx, aeroCluster)
			Expect(err).ToNot(HaveOccurred())
		},
	)

	AfterEach(
		func() {
			aeroCluster, err := getCluster(
				k8sClient, ctx, clusterNamespacedName,
			)
			Expect(err).ToNot(HaveOccurred())

			_ = deleteCluster(k8sClient, ctx, aeroCluster)
		},
	)

	Context(
		"When doing valid operations", func() {
			It(
				"Try update operations", func() {
					By("ScaleUp")

					err := scaleUpClusterTest(
						k8sClient, ctx, clusterNamespacedName, 1,
					)
					Expect(err).ToNot(HaveOccurred())

					By("ScaleDown")

					// TODO:
					// How to check if it is checking cluster stability before killing node
					// Check if tip-clear, alumni-reset is done or not
					err = scaleDownClusterTest(
						k8sClient, ctx, clusterNamespacedName, 1,
					)
					Expect(err).ToNot(HaveOccurred())

					By("RollingRestart")

					// TODO: How to check if it is checking cluster stability before killing node
					err = rollingRestartClusterTest(
						logger, k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					By("RollingRestart By Updating NamespaceStorage")

					err = rollingRestartClusterByUpdatingNamespaceStorageTest(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					By("RollingRestart By Adding Namespace Dynamically")

					err = rollingRestartClusterByAddingNamespaceDynamicallyTest(
						k8sClient, ctx, dynamicNs, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					By("Scaling up along with modifying Namespace storage Dynamically")

					err = scaleUpClusterTestWithNSDeviceHandling(
						k8sClient, ctx, clusterNamespacedName, 1,
					)
					Expect(err).ToNot(HaveOccurred())

					By("Scaling down along with modifying Namespace storage Dynamically")

					err = scaleDownClusterTestWithNSDeviceHandling(
						k8sClient, ctx, clusterNamespacedName, 1,
					)
					Expect(err).ToNot(HaveOccurred())

					By("RollingRestart By Removing Namespace Dynamically")

					err = rollingRestartClusterByRemovingNamespaceDynamicallyTest(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					By("RollingRestart By Re-using Previously Removed Namespace Storage")

					err = rollingRestartClusterByAddingNamespaceDynamicallyTest(
						k8sClient, ctx, dynamicNs, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					By("Upgrade/Downgrade")

					// TODO: How to check if it is checking cluster stability before killing node
					// dont change image, it upgrade, check old version
					err = upgradeClusterTest(
						k8sClient, ctx, clusterNamespacedName, prevImage,
					)
					Expect(err).ToNot(HaveOccurred())
				},
			)
		},
	)

	Context(
		"When doing invalid operations", func() {

			Context(
				"ValidateUpdate", func() {
					// TODO: No jump version yet but will be used
					// It("Image", func() {
					// 	old := aeroCluster.Spec.Image
					// 	aeroCluster.Spec.Image = "aerospike/aerospike-server-enterprise:4.0.0.5"
					// 	err = k8sClient.Update(ctx, aeroCluster)
					// 	validateError(err, "should fail for upgrading to jump version")
					// 	aeroCluster.Spec.Image = old
					// })
					It(
						"MultiPodPerHost: should fail for updating MultiPodPerHost. Cannot be updated",
						func() {
							aeroCluster, err := getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())

							aeroCluster.Spec.PodSpec.MultiPodPerHost = !aeroCluster.Spec.PodSpec.MultiPodPerHost

							err = k8sClient.Update(ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)

					It(
						"Should fail for Re-using Namespace Storage Dynamically",
						func() {
							err := rollingRestartClusterByReusingNamespaceStorageTest(
								k8sClient, ctx, clusterNamespacedName, dynamicNs,
							)
							Expect(err).Should(HaveOccurred())
						},
					)

					It(
						"StorageValidation: should fail for updating Storage. Cannot be updated",
						func() {
							aeroCluster, err := getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())

							newVolumeSpec := []asdbv1beta1.VolumeSpec{
								{
									Name: "ns",
									Source: asdbv1beta1.VolumeSource{
										PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
											StorageClass: storageClass,
											VolumeMode:   v1.PersistentVolumeBlock,
											Size:         resource.MustParse("1Gi"),
										},
									},
									Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
										Path: "/dev/xvdf2",
									},
								},
								{
									Name: "workdir",
									Source: asdbv1beta1.VolumeSource{
										PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
											StorageClass: storageClass,
											VolumeMode:   v1.PersistentVolumeFilesystem,
											Size:         resource.MustParse("1Gi"),
										},
									},
									Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
										Path: "/opt/aeropsike/ns1",
									},
								},
							}
							aeroCluster.Spec.Storage.Volumes = newVolumeSpec

							err = k8sClient.Update(ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)

					Context(
						"AerospikeConfig", func() {
							Context(
								"Namespace", func() {
									It(
										"UpdateReplicationFactor: should fail for updating namespace replication-factor. Cannot be updated",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["replication-factor"] = 5

											err = k8sClient.Update(
												ctx, aeroCluster,
											)
											Expect(err).Should(HaveOccurred())
										},
									)
								},
							)
						},
					)
				},
			)
		},
	)
}

// Test cluster validation Common for deployment and update both
func NegativeClusterValidationTest(ctx goctx.Context) {

	clusterName := "invalid-cluster"
	clusterNamespacedName := getClusterNamespacedName(clusterName, namespace)

	Context(
		"NegativeDeployClusterValidationTest", func() {
			negativeDeployClusterValidationTest(ctx, clusterNamespacedName)
		},
	)

	Context(
		"NegativeUpdateClusterValidationTest", func() {
			negativeUpdateClusterValidationTest(ctx, clusterNamespacedName)
		},
	)
}

func negativeDeployClusterValidationTest(
	ctx goctx.Context, clusterNamespacedName types.NamespacedName,
) {
	Context(
		"Validation", func() {

			It(
				"EmptyClusterName: should fail for EmptyClusterName", func() {
					cName := getClusterNamespacedName(
						"", clusterNamespacedName.Namespace,
					)

					aeroCluster := createDummyAerospikeCluster(cName, 1)
					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)

			It(
				"EmptyNamespaceName: should fail for EmptyNamespaceName",
				func() {
					cName := getClusterNamespacedName("validclustername", "")

					aeroCluster := createDummyAerospikeCluster(cName, 1)
					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)

			It(
				"InvalidImage: should fail for InvalidImage", func() {
					aeroCluster := createDummyAerospikeCluster(
						clusterNamespacedName, 1,
					)
					aeroCluster.Spec.Image = "InvalidImage"
					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())

					aeroCluster.Spec.Image = invalidImage
					err = deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)

			It(
				"InvalidSize: should fail for zero size", func() {
					aeroCluster := createDummyAerospikeCluster(
						clusterNamespacedName, 0,
					)
					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())

					// aeroCluster = createDummyAerospikeCluster(clusterNamespacedName, 9)
					// err = deployCluster(k8sClient, ctx, aeroCluster)
					// validateError(err, "should fail for community eidition having more than 8 nodes")
				},
			)

			Context(
				"InvalidAerospikeConfig: should fail for empty/invalid aerospikeConfig",
				func() {
					It(
						"should fail for empty/invalid aerospikeConfig",
						func() {
							aeroCluster := createDummyAerospikeCluster(
								clusterNamespacedName, 1,
							)
							aeroCluster.Spec.AerospikeConfig = &asdbv1beta1.AerospikeConfigSpec{}
							err := deployCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())

							aeroCluster = createDummyAerospikeCluster(
								clusterNamespacedName, 1,
							)
							aeroCluster.Spec.AerospikeConfig = &asdbv1beta1.AerospikeConfigSpec{
								Value: map[string]interface{}{
									"namespaces": "invalidConf",
								},
							}
							err = deployCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())

						},
					)

					Context(
						"InvalidNamespace", func() {
							It(
								"NilAerospikeNamespace: should fail for nil aerospikeConfig.namespace",
								func() {
									aeroCluster := createDummyAerospikeCluster(
										clusterNamespacedName, 1,
									)
									aeroCluster.Spec.AerospikeConfig.Value["namespaces"] = nil
									err := deployCluster(
										k8sClient, ctx, aeroCluster,
									)
									Expect(err).Should(HaveOccurred())
								},
							)

							// Should we test for overridden fields
							Context(
								"InvalidStorage", func() {
									It(
										"NilStorageEngine: should fail for nil storage-engine",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"] = nil
											err := deployCluster(
												k8sClient, ctx, aeroCluster,
											)
											Expect(err).Should(HaveOccurred())
										},
									)

									It(
										"NilStorageEngineDevice: should fail for nil storage-engine.device",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"] = nil
												err := deployCluster(
													k8sClient, ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"InvalidStorageEngineDevice: should fail for invalid storage-engine.device, cannot have 3 devices in single device string",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												aeroCluster.Spec.Storage.Volumes = []asdbv1beta1.VolumeSpec{
													{
														Name: "nsvol1",
														Source: asdbv1beta1.VolumeSource{
															PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
																Size:         resource.MustParse("1Gi"),
																StorageClass: storageClass,
																VolumeMode:   v1.PersistentVolumeBlock,
															},
														},
														Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
															Path: "/dev/xvdf1",
														},
													},
													{
														Name: "nsvol2",
														Source: asdbv1beta1.VolumeSource{
															PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
																Size:         resource.MustParse("1Gi"),
																StorageClass: storageClass,
																VolumeMode:   v1.PersistentVolumeBlock,
															},
														},
														Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
															Path: "/dev/xvdf2",
														},
													},
													{
														Name: "nsvol3",
														Source: asdbv1beta1.VolumeSource{
															PersistentVolume: &asdbv1beta1.PersistentVolumeSpec{
																Size:         resource.MustParse("1Gi"),
																StorageClass: storageClass,
																VolumeMode:   v1.PersistentVolumeBlock,
															},
														},
														Aerospike: &asdbv1beta1.AerospikeServerVolumeAttachment{
															Path: "/dev/xvdf3",
														},
													},
												}

												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"] = []string{"/dev/xvdf1 /dev/xvdf2 /dev/xvdf3"}
												err := deployCluster(
													k8sClient, ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"NilStorageEngineFile: should fail for nil storage-engine.file",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["files"]; ok {
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["files"] = nil
												err := deployCluster(
													k8sClient, ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"ExtraStorageEngineDevice: should fail for invalid storage-engine.device, cannot use a device which doesn't exist in storage",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												devList := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"].([]interface{})
												devList = append(
													devList, "andRandomDevice",
												)
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"] = devList
												err := deployCluster(
													k8sClient, ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"DuplicateStorageEngineDevice: should fail for invalid storage-engine.device, cannot use a device which already exist in another namespace",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											secondNs := map[string]interface{}{
												"name":               "ns1",
												"memory-size":        1000955200,
												"replication-factor": 2,
												"storage-engine": map[string]interface{}{
													"type":    "device",
													"devices": []interface{}{"/test/dev/xvdf"},
												},
											}

											nsList := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})
											nsList = append(nsList, secondNs)
											aeroCluster.Spec.AerospikeConfig.Value["namespaces"] = nsList
											err := deployCluster(
												k8sClient, ctx, aeroCluster,
											)
											Expect(err).Should(HaveOccurred())
										},
									)

									It(
										"InvalidxdrConfig: should fail for invalid xdr config. mountPath for digestlog not present in storage",
										func() {
											aeroCluster := createDummyAerospikeCluster(
												clusterNamespacedName, 1,
											)
											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												aeroCluster.Spec.Storage = asdbv1beta1.AerospikeStorageSpec{}
												aeroCluster.Spec.AerospikeConfig.Value["xdr"] = map[string]interface{}{
													"enable-xdr":         false,
													"xdr-digestlog-path": "/opt/aerospike/xdr/digestlog 100G",
												}
												err := deployCluster(
													k8sClient, ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)
								},
							)
						},
					)

					Context(
						"ChangeDefaultConfig", func() {
							It(
								"NsConf", func() {
									// Ns conf
									// Rack-id
									// aeroCluster := createDummyAerospikeCluster(clusterNamespacedName, 1)
									// aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["rack-id"] = 1
									// aeroCluster.Spec.RackConfig.Namespaces = []string{"test"}
									// err := deployCluster(k8sClient, ctx, aeroCluster)
									// validateError(err, "should fail for setting rack-id")
								},
							)

							It(
								"ServiceConf: should fail for setting node-id/cluster-name",
								func() {
									// Service conf
									// 	"node-id"
									// 	"cluster-name"
									aeroCluster := createDummyAerospikeCluster(
										clusterNamespacedName, 1,
									)
									aeroCluster.Spec.AerospikeConfig.Value["service"].(map[string]interface{})["node-id"] = "a1"
									err := deployCluster(
										k8sClient, ctx, aeroCluster,
									)
									Expect(err).Should(HaveOccurred())

									aeroCluster = createDummyAerospikeCluster(
										clusterNamespacedName, 1,
									)
									aeroCluster.Spec.AerospikeConfig.Value["service"].(map[string]interface{})["cluster-name"] = "cluster-name"
									err = deployCluster(
										k8sClient, ctx, aeroCluster,
									)
									Expect(err).Should(HaveOccurred())
								},
							)

							It(
								"NetworkConf: should fail for setting network conf/tls network conf",
								func() {
									// Network conf
									// "port"
									// "access-port"
									// "access-addresses"
									// "alternate-access-port"
									// "alternate-access-addresses"
									aeroCluster := createDummyAerospikeCluster(
										clusterNamespacedName, 1,
									)
									networkConf := map[string]interface{}{
										"service": map[string]interface{}{
											"port":             3000,
											"access-addresses": []string{"<access_addresses>"},
										},
									}
									aeroCluster.Spec.AerospikeConfig.Value["network"] = networkConf
									err := deployCluster(
										k8sClient, ctx, aeroCluster,
									)
									Expect(err).Should(HaveOccurred())

									// if "tls-name" in conf
									// "tls-port"
									// "tls-access-port"
									// "tls-access-addresses"
									// "tls-alternate-access-port"
									// "tls-alternate-access-addresses"
									aeroCluster = createDummyAerospikeCluster(
										clusterNamespacedName, 1,
									)
									networkConf = map[string]interface{}{
										"service": map[string]interface{}{
											"tls-name":             "aerospike-a-0.test-runner",
											"tls-port":             3001,
											"tls-access-addresses": []string{"<tls-access-addresses>"},
										},
									}
									aeroCluster.Spec.AerospikeConfig.Value["network"] = networkConf
									err = deployCluster(
										k8sClient, ctx, aeroCluster,
									)
									Expect(err).Should(HaveOccurred())
								},
							)

							// Logging conf
							// XDR conf
						},
					)
				},
			)

			Context(
				"InvalidAerospikeConfigSecret", func() {
					It(
						"WhenFeatureKeyExist: should fail for no feature-key-file path in storage volume",
						func() {
							aeroCluster := createAerospikeClusterPost560(
								clusterNamespacedName, 1, latestImage,
							)
							aeroCluster.Spec.AerospikeConfig.Value["service"] = map[string]interface{}{
								"feature-key-file": "/randompath/features.conf",
							}
							err := deployCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)

					It(
						"WhenTLSExist: should fail for no tls path in storage volume",
						func() {
							aeroCluster := createAerospikeClusterPost560(
								clusterNamespacedName, 1, latestImage,
							)
							aeroCluster.Spec.AerospikeConfig.Value["network"] = map[string]interface{}{
								"tls": []interface{}{
									map[string]interface{}{
										"name":      "aerospike-a-0.test-runner",
										"cert-file": "/randompath/svc_cluster_chain.pem",
									},
								},
							}
							err := deployCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)
				},
			)

			Context(
				"InvalidDNSConfiguration", func() {
					It(
						"InvalidDnsPolicy: should fail when dnsPolicy is set to 'Default'",
						func() {
							aeroCluster := createDummyAerospikeCluster(clusterNamespacedName, 2)
							defaultDNS := v1.DNSDefault
							aeroCluster.Spec.PodSpec.InputDNSPolicy = &defaultDNS
							err := deployCluster(k8sClient, ctx, aeroCluster)

							Expect(err).Should(HaveOccurred())
						},
					)

					It(
						"MissingDnsConfig: should fail when dnsPolicy is set to 'None' and no dnsConfig given",
						func() {
							aeroCluster := createDummyAerospikeCluster(clusterNamespacedName, 2)
							noneDNS := v1.DNSNone
							aeroCluster.Spec.PodSpec.InputDNSPolicy = &noneDNS
							err := deployCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)
				},
			)
		},
	)
}

func negativeUpdateClusterValidationTest(
	ctx goctx.Context, clusterNamespacedName types.NamespacedName,
) {
	// Will be used in Update

	Context(
		"Validation", func() {
			BeforeEach(
				func() {
					aeroCluster := createDummyAerospikeCluster(
						clusterNamespacedName, 3,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
				},
			)

			AfterEach(
				func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					_ = deleteCluster(k8sClient, ctx, aeroCluster)
				},
			)

			It(
				"InvalidImage: should fail for InvalidImage, should fail for image lower than base",
				func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.Image = "InvalidImage"
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())

					aeroCluster, err = getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.Image = invalidImage
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)

			It(
				"InvalidSize: should fail for zero size", func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.Size = 0
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())

					// aeroCluster = createDummyAerospikeCluster(clusterNamespacedName, 9)
					// err = deployCluster(k8sClient, ctx, aeroCluster)
					// validateError(err, "should fail for community eidition having more than 8 nodes")
				},
			)

			Context(
				"InvalidDNSConfiguration", func() {
					It(
						"InvalidDnsPolicy: should fail when dnsPolicy is set to 'Default'",
						func() {
							aeroCluster, err := getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())
							defaultDNS := v1.DNSDefault
							aeroCluster.Spec.PodSpec.InputDNSPolicy = &defaultDNS
							err = updateCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)

					It(
						"MissingDnsConfig: Should fail when dnsPolicy is set to 'None' and no dnsConfig given",
						func() {
							aeroCluster, err := getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())
							noneDNS := v1.DNSNone
							aeroCluster.Spec.PodSpec.InputDNSPolicy = &noneDNS
							err = updateCluster(k8sClient, ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())
						},
					)
				},
			)

			Context(
				"InvalidAerospikeConfig: should fail for empty aerospikeConfig, should fail for invalid aerospikeConfig",
				func() {
					It(
						"should fail for empty aerospikeConfig, should fail for invalid aerospikeConfig",
						func() {
							aeroCluster, err := getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())

							aeroCluster.Spec.AerospikeConfig = &asdbv1beta1.AerospikeConfigSpec{}
							err = k8sClient.Update(ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())

							aeroCluster, err = getCluster(
								k8sClient, ctx, clusterNamespacedName,
							)
							Expect(err).ToNot(HaveOccurred())

							aeroCluster.Spec.AerospikeConfig = &asdbv1beta1.AerospikeConfigSpec{
								Value: map[string]interface{}{
									"namespaces": "invalidConf",
								},
							}
							err = k8sClient.Update(ctx, aeroCluster)
							Expect(err).Should(HaveOccurred())

						},
					)

					Context(
						"InvalidNamespace", func() {
							It(
								"NilAerospikeNamespace: should fail for nil aerospikeConfig.namespace",
								func() {
									aeroCluster, err := getCluster(
										k8sClient, ctx, clusterNamespacedName,
									)
									Expect(err).ToNot(HaveOccurred())

									aeroCluster.Spec.AerospikeConfig.Value["namespaces"] = nil
									err = k8sClient.Update(ctx, aeroCluster)
									Expect(err).Should(HaveOccurred())
								},
							)

							// Should we test for overridden fields
							Context(
								"InvalidStorage", func() {
									It(
										"NilStorageEngine: should fail for nil storage-engine",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"] = nil
											err = k8sClient.Update(
												ctx, aeroCluster,
											)
											Expect(err).Should(HaveOccurred())
										},
									)

									It(
										"NilStorageEngineDevice: should fail for nil storage-engine.device",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"] = nil
												err = k8sClient.Update(
													ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"NilStorageEngineFile: should fail for nil storage-engine.file",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["files"]; ok {
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["files"] = nil
												err = k8sClient.Update(
													ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"ExtraStorageEngineDevice: should fail for invalid storage-engine.device, cannot add a device which doesn't exist in BlockStorage",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												devList := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"].([]interface{})
												devList = append(
													devList, "andRandomDevice",
												)
												aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"] = devList
												err = k8sClient.Update(
													ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)

									It(
										"DuplicateStorageEngineDevice: should fail for invalid storage-engine.device, cannot add a device which already exist in another namespace",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											secondNs := map[string]interface{}{
												"name":               "ns1",
												"memory-size":        1000955200,
												"replication-factor": 2,
												"storage-engine": map[string]interface{}{
													"type":    "device",
													"devices": []interface{}{"/test/dev/xvdf"},
												},
											}

											nsList := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})
											nsList = append(nsList, secondNs)
											aeroCluster.Spec.AerospikeConfig.Value["namespaces"] = nsList
											err = k8sClient.Update(
												ctx, aeroCluster,
											)
											Expect(err).Should(HaveOccurred())
										},
									)

									It(
										"InvalidxdrConfig: should fail for invalid xdr config. mountPath for digestlog not present in fileStorage",
										func() {
											aeroCluster, err := getCluster(
												k8sClient, ctx,
												clusterNamespacedName,
											)
											Expect(err).ToNot(HaveOccurred())

											if _, ok := aeroCluster.Spec.AerospikeConfig.Value["namespaces"].([]interface{})[0].(map[string]interface{})["storage-engine"].(map[string]interface{})["devices"]; ok {
												aeroCluster.Spec.AerospikeConfig.Value["xdr"] = map[string]interface{}{
													"enable-xdr":         false,
													"xdr-digestlog-path": "randomPath 100G",
												}
												err = k8sClient.Update(
													ctx, aeroCluster,
												)
												Expect(err).Should(HaveOccurred())
											}
										},
									)
								},
							)
						},
					)

					Context(
						"ChangeDefaultConfig", func() {

							It(
								"ServiceConf: should fail for setting node-id, should fail for setting cluster-name",
								func() {
									// Service conf
									// 	"node-id"
									// 	"cluster-name"
									aeroCluster, err := getCluster(
										k8sClient, ctx, clusterNamespacedName,
									)
									Expect(err).ToNot(HaveOccurred())

									aeroCluster.Spec.AerospikeConfig.Value["service"].(map[string]interface{})["node-id"] = "a10"
									err = k8sClient.Update(ctx, aeroCluster)
									Expect(err).Should(HaveOccurred())

									aeroCluster, err = getCluster(
										k8sClient, ctx, clusterNamespacedName,
									)
									Expect(err).ToNot(HaveOccurred())

									aeroCluster.Spec.AerospikeConfig.Value["service"].(map[string]interface{})["cluster-name"] = "cluster-name"
									err = k8sClient.Update(ctx, aeroCluster)
									Expect(err).Should(HaveOccurred())
								},
							)

							It(
								"NetworkConf: should fail for setting network conf, should fail for setting tls network conf",
								func() {
									// Network conf
									// "port"
									// "access-port"
									// "access-addresses"
									// "alternate-access-port"
									// "alternate-access-addresses"
									aeroCluster, err := getCluster(
										k8sClient, ctx, clusterNamespacedName,
									)
									Expect(err).ToNot(HaveOccurred())

									networkConf := map[string]interface{}{
										"service": map[string]interface{}{
											"port":             3000,
											"access-addresses": []string{"<access_addresses>"},
										},
									}
									aeroCluster.Spec.AerospikeConfig.Value["network"] = networkConf
									err = k8sClient.Update(ctx, aeroCluster)
									Expect(err).Should(HaveOccurred())

									// if "tls-name" in conf
									// "tls-port"
									// "tls-access-port"
									// "tls-access-addresses"
									// "tls-alternate-access-port"
									// "tls-alternate-access-addresses"
									aeroCluster, err = getCluster(
										k8sClient, ctx, clusterNamespacedName,
									)
									Expect(err).ToNot(HaveOccurred())

									networkConf = map[string]interface{}{
										"service": map[string]interface{}{
											"tls-name":             "aerospike-a-0.test-runner",
											"tls-port":             3001,
											"tls-access-addresses": []string{"<tls-access-addresses>"},
										},
									}
									aeroCluster.Spec.AerospikeConfig.Value["network"] = networkConf
									err = k8sClient.Update(ctx, aeroCluster)
									Expect(err).Should(HaveOccurred())
								},
							)

							// Logging conf
							// XDR conf
						},
					)
				},
			)
		},
	)

	Context(
		"InvalidAerospikeConfigSecret", func() {
			BeforeEach(
				func() {
					aeroCluster := createAerospikeClusterPost560(
						clusterNamespacedName, 2, latestImage,
					)

					err := deployCluster(k8sClient, ctx, aeroCluster)
					Expect(err).ToNot(HaveOccurred())
				},
			)

			AfterEach(
				func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					_ = deleteCluster(k8sClient, ctx, aeroCluster)
				},
			)

			It(
				"WhenFeatureKeyExist: should fail for no feature-key-file path in storage volumes",
				func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.AerospikeConfig.Value["service"] = map[string]interface{}{
						"feature-key-file": "/randompath/features.conf",
					}
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)

			It(
				"WhenTLSExist: should fail for no tls path in storage voluems",
				func() {
					aeroCluster, err := getCluster(
						k8sClient, ctx, clusterNamespacedName,
					)
					Expect(err).ToNot(HaveOccurred())

					aeroCluster.Spec.AerospikeConfig.Value["network"] = map[string]interface{}{
						"tls": []interface{}{
							map[string]interface{}{
								"name":      "aerospike-a-0.test-runner",
								"cert-file": "/randompath/svc_cluster_chain.pem",
							},
						},
					}
					err = k8sClient.Update(ctx, aeroCluster)
					Expect(err).Should(HaveOccurred())
				},
			)
		},
	)
}
