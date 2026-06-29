package troubleshoot

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/kubevirt"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

type TroubleshootToolSuite struct {
	suite.Suite
}

func (s *TroubleshootToolSuite) TestToolRegistration() {
	s.Run("tool is registered", func() {
		tools := Tools()
		s.Require().Len(tools, 1, "Expected 1 troubleshoot tool")
		s.Equal("vm_troubleshoot", tools[0].Tool.Name)
		s.Equal("Virtual Machine: Troubleshoot", tools[0].Tool.Annotations.Title)
		s.NotNil(tools[0].Tool.InputSchema)
		s.NotNil(tools[0].Handler)
	})

	s.Run("tool has correct annotations", func() {
		tools := Tools()
		tool := tools[0].Tool

		s.True(*tool.Annotations.ReadOnlyHint, "troubleshoot should be read-only")
		s.False(*tool.Annotations.DestructiveHint, "troubleshoot should not be destructive")
		s.True(*tool.Annotations.IdempotentHint, "troubleshoot should be idempotent")
		s.True(*tool.Annotations.OpenWorldHint, "troubleshoot should be open-world")
	})

	s.Run("tool has correct schema", func() {
		tools := Tools()
		schema := tools[0].Tool.InputSchema

		s.Require().NotNil(schema.Properties)
		s.Contains(schema.Properties, "namespace")
		s.Contains(schema.Properties, "name")
		s.ElementsMatch([]string{"namespace", "name"}, schema.Required)
	})

	s.Run("description mentions priority and scope", func() {
		tools := Tools()
		desc := tools[0].Tool.Description
		s.Contains(desc, "FIRST")
		s.Contains(desc, "root-cause")
		s.Contains(desc, "StorageClasses")
		s.Contains(desc, "cloud-init")
	})
}

func (s *TroubleshootToolSuite) TestFetchVMStatus() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		kubevirt.VirtualMachineGVR: "VirtualMachineList",
	}

	s.Run("returns status when VM exists", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"status": map[string]interface{}{
				"printableStatus": "Running",
				"ready":           true,
			},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, testVM)
		result, vm := fetchVMStatus(ctx, client, "test-ns", "test-vm")

		s.Contains(result, "## VirtualMachine Status: test-ns/test-vm")
		s.Contains(result, "printableStatus")
		s.Contains(result, "Running")
		s.NotNil(vm)
	})

	s.Run("returns error when VM not found", func() {
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result, vm := fetchVMStatus(ctx, client, "test-ns", "nonexistent")

		s.Contains(result, "## VirtualMachine")
		s.Contains(result, "Error")
		s.Nil(vm)
	})

	s.Run("handles VM with no status field", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, testVM)
		result, vm := fetchVMStatus(ctx, client, "test-ns", "test-vm")

		s.Contains(result, "No status found")
		s.NotNil(vm)
	})
}

func (s *TroubleshootToolSuite) TestFetchVMIStatus() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		kubevirt.VirtualMachineInstanceGVR: "VirtualMachineInstanceList",
	}

	s.Run("returns status when VMI exists", func() {
		testVMI := &unstructured.Unstructured{}
		testVMI.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachineInstance",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"status": map[string]interface{}{
				"phase":    "Running",
				"nodeName": "worker-1",
			},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, testVMI)
		result, vmi := fetchVMIStatus(ctx, client, "test-ns", "test-vm")

		s.Contains(result, "## VirtualMachineInstance Status: test-ns/test-vm")
		s.Contains(result, "phase")
		s.Contains(result, "Running")
		s.NotNil(vmi)
	})

	s.Run("returns info message when VMI not found", func() {
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result, vmi := fetchVMIStatus(ctx, client, "test-ns", "nonexistent")

		s.Contains(result, "## VirtualMachineInstance")
		s.Contains(result, "not found")
		s.Nil(vmi)
	})
}

func (s *TroubleshootToolSuite) TestFetchVolumes() {
	s.Run("returns volumes from VM spec", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "rootdisk",
								"containerDisk": map[string]interface{}{
									"image": "quay.io/containerdisks/fedora:latest",
								},
							},
							map[string]interface{}{
								"name": "cloudinitdisk",
								"cloudInitNoCloud": map[string]interface{}{
									"userData": "#cloud-config\nruncmd:\n  - shutdown -h now",
								},
							},
						},
					},
				},
			},
		})

		result := fetchVolumes("test-ns", "test-vm", testVM, nil)

		s.Contains(result, "## Volumes")
		s.Contains(result, "VirtualMachine")
		s.Contains(result, "rootdisk")
		s.Contains(result, "cloudinitdisk")
	})

	s.Run("falls back to VMI when VM has no volumes", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
		})

		testVMI := &unstructured.Unstructured{}
		testVMI.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachineInstance",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "rootdisk",
						"containerDisk": map[string]interface{}{
							"image": "quay.io/containerdisks/fedora:latest",
						},
					},
				},
			},
		})

		result := fetchVolumes("test-ns", "test-vm", testVM, testVMI)

		s.Contains(result, "## Volumes")
		s.Contains(result, "VirtualMachineInstance")
		s.Contains(result, "rootdisk")
	})

	s.Run("returns no volumes message when both nil", func() {
		result := fetchVolumes("test-ns", "test-vm", nil, nil)
		s.Contains(result, "No volumes configured")
	})

	s.Run("returns no volumes when spec is empty", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
		})

		result := fetchVolumes("test-ns", "test-vm", testVM, nil)
		s.Contains(result, "No volumes configured")
	})
}

func (s *TroubleshootToolSuite) TestFetchVirtLauncherPod() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		kubevirt.PodGVR: "PodList",
	}

	s.Run("returns pod info when found", func() {
		testPod := &unstructured.Unstructured{}
		testPod.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "virt-launcher-test-vm-abc123",
				"namespace": "test-ns",
				"labels": map[string]interface{}{
					"kubevirt.io":         "virt-launcher",
					"vm.kubevirt.io/name": "test-vm",
				},
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, testPod)
		result, podNames := fetchVirtLauncherPod(ctx, client, "test-ns", "test-vm")

		s.Contains(result, "## virt-launcher Pod")
		s.Contains(result, "virt-launcher-test-vm-abc123")
		s.Require().Len(podNames, 1)
		s.Equal("virt-launcher-test-vm-abc123", podNames[0])
	})

	s.Run("returns message when no pod found", func() {
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result, podNames := fetchVirtLauncherPod(ctx, client, "test-ns", "test-vm")

		s.Contains(result, "No virt-launcher pod found")
		s.Nil(podNames)
	})
}

func (s *TroubleshootToolSuite) TestFetchVirtLauncherPodLogs() {
	s.Run("returns message when no pod names", func() {
		result := fetchVirtLauncherPodLogs(context.Background(), nil, "test-ns", nil)
		s.Contains(result, "## virt-launcher Pod Logs")
		s.Contains(result, "No pod found")
	})

	s.Run("returns message when empty pod names", func() {
		result := fetchVirtLauncherPodLogs(context.Background(), nil, "test-ns", []string{})
		s.Contains(result, "No pod found")
	})
}

func (s *TroubleshootToolSuite) TestFetchDataVolumeStatus() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		kubevirt.DataVolumeGVR:            "DataVolumeList",
		kubevirt.PersistentVolumeClaimGVR: "PersistentVolumeClaimList",
	}

	s.Run("returns DV status when DataVolume exists", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"dataVolumeTemplates": []interface{}{
					map[string]interface{}{
						"metadata": map[string]interface{}{
							"name": "test-vm-volume",
						},
						"spec": map[string]interface{}{
							"storage": map[string]interface{}{
								"storageClassName": "premium-storage",
							},
						},
					},
				},
			},
		})

		testDV := &unstructured.Unstructured{}
		testDV.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "cdi.kubevirt.io/v1beta1",
			"kind":       "DataVolume",
			"metadata": map[string]interface{}{
				"name":      "test-vm-volume",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClassName": "premium-storage",
				},
			},
			"status": map[string]interface{}{
				"phase": "Succeeded",
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, testDV)
		result := fetchDataVolumeStatus(ctx, client, "test-ns", testVM)

		s.Contains(result, "## DataVolume/PVC Status")
		s.Contains(result, "test-vm-volume")
		s.Contains(result, "premium-storage")
		s.Contains(result, "Succeeded")
	})

	s.Run("returns not found when DataVolume missing", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"dataVolumeTemplates": []interface{}{
					map[string]interface{}{
						"metadata": map[string]interface{}{
							"name": "missing-volume",
						},
					},
				},
			},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result := fetchDataVolumeStatus(ctx, client, "test-ns", testVM)

		s.Contains(result, "## DataVolume/PVC Status")
		s.Contains(result, "missing-volume")
		s.Contains(result, "not found")
	})

	s.Run("returns message when VM is nil", func() {
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result := fetchDataVolumeStatus(ctx, client, "test-ns", nil)

		s.Contains(result, "No VM available")
	})

	s.Run("returns message when no dataVolumeTemplates", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{},
		})

		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result := fetchDataVolumeStatus(ctx, client, "test-ns", testVM)

		s.Contains(result, "No dataVolumeTemplates")
	})
}

func (s *TroubleshootToolSuite) TestExtractCloudInit() {
	s.Run("extracts cloudInitNoCloud userData with runcmd visible", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "cloudinitdisk",
								"cloudInitNoCloud": map[string]interface{}{
									"userData": "#cloud-config\nruncmd:\n  - [\"shutdown\", \"-h\", \"now\"]",
								},
							},
						},
					},
				},
			},
		})

		result := extractCloudInit(testVM, nil)

		s.Contains(result, "## Cloud-Init Configuration")
		s.Contains(result, "cloudinitdisk")
		s.Contains(result, "cloudInitNoCloud")
		s.Contains(result, "shutdown")
	})

	s.Run("redacts sensitive fields in userData", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "cloudinitdisk",
								"cloudInitNoCloud": map[string]interface{}{
									"userData": "#cloud-config\npassword: supersecret123\nruncmd:\n  - echo hello\nssh_authorized_keys:\n  - ssh-rsa AAAA...",
								},
							},
						},
					},
				},
			},
		})

		result := extractCloudInit(testVM, nil)

		s.Contains(result, "password: <REDACTED>")
		s.NotContains(result, "supersecret123")
		s.Contains(result, "runcmd")
		s.Contains(result, "echo hello")
		s.Contains(result, "ssh_authorized_keys: <REDACTED>")
	})

	s.Run("returns no cloud-init message when none configured", func() {
		testVM := &unstructured.Unstructured{}
		testVM.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "rootdisk",
								"containerDisk": map[string]interface{}{
									"image": "quay.io/containerdisks/fedora:latest",
								},
							},
						},
					},
				},
			},
		})

		result := extractCloudInit(testVM, nil)

		s.Contains(result, "No cloud-init volumes configured")
	})

	s.Run("returns no volumes message when both nil", func() {
		result := extractCloudInit(nil, nil)
		s.Contains(result, "No volumes found")
	})

	s.Run("extracts from VMI when VM has no volumes", func() {
		testVMI := &unstructured.Unstructured{}
		testVMI.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachineInstance",
			"metadata": map[string]interface{}{
				"name":      "test-vm",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "cloudinitdisk",
						"cloudInitConfigDrive": map[string]interface{}{
							"userData": "#cloud-config\nruncmd:\n  - echo hello",
						},
					},
				},
			},
		})

		result := extractCloudInit(nil, testVMI)

		s.Contains(result, "## Cloud-Init Configuration")
		s.Contains(result, "cloudInitConfigDrive")
		s.Contains(result, "echo hello")
	})
}

func TestTroubleshootToolSuite(t *testing.T) {
	suite.Run(t, new(TroubleshootToolSuite))
}
