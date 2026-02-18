package mustgather

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes/fakeclient"
	"github.com/stretchr/testify/require"
)

func TestPlanMustGather(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		params           PlanMustGatherParams
		shouldContain    []string
		shouldNotContain []string
		wantError        string
	}{
		{
			name:   "generates plan with default values",
			params: PlanMustGatherParams{},
			shouldContain: []string{
				"apiVersion: v1",
				"kind: Pod",
				"kind: ServiceAccount",
				"kind: ClusterRoleBinding",
				"must-gather-collector",
				"image: registry.redhat.io/openshift4/ose-must-gather:latest",
				"mountPath: /must-gather",
			},
		},
		{
			name:          "generates plan with custom namespace",
			params:        PlanMustGatherParams{Namespace: "custom-must-gather-ns"},
			shouldContain: []string{"namespace: custom-must-gather-ns"},
		},
		{
			name:          "generates plan with node name",
			params:        PlanMustGatherParams{NodeName: "worker-node-1"},
			shouldContain: []string{"nodeName: worker-node-1"},
		},
		{
			name:          "generates plan with host network enabled",
			params:        PlanMustGatherParams{HostNetwork: true},
			shouldContain: []string{"hostNetwork: true"},
		},
		{
			name:          "generates plan with custom source dir",
			params:        PlanMustGatherParams{SourceDir: "/custom/gather/path"},
			shouldContain: []string{"mountPath: /custom/gather/path"},
		},
		{
			name: "generates plan with multiple custom images",
			params: PlanMustGatherParams{
				Images: []string{"quay.io/custom/must-gather-1:v1", "quay.io/custom/must-gather-2:v2"},
			},
			shouldContain: []string{
				"image: quay.io/custom/must-gather-1:v1",
				"image: quay.io/custom/must-gather-2:v2",
				"name: gather-1",
				"name: gather-2",
			},
		},
		{
			name: "returns error when more than eight images",
			params: PlanMustGatherParams{
				Images: []string{
					"quay.io/image/1", "quay.io/image/2", "quay.io/image/3", "quay.io/image/4",
					"quay.io/image/5", "quay.io/image/6", "quay.io/image/7", "quay.io/image/8",
					"quay.io/image/9",
				},
			},
			wantError: "more than 8 gather images are not supported",
		},
		{
			name:          "generates plan with valid timeout",
			params:        PlanMustGatherParams{Timeout: "30m"},
			shouldContain: []string{"/usr/bin/timeout 30m /usr/bin/gather"},
		},
		{
			name:      "returns error for invalid timeout format",
			params:    PlanMustGatherParams{Timeout: "invalid-duration"},
			wantError: "timeout duration is not valid",
		},
		{
			name:          "generates plan with valid since duration",
			params:        PlanMustGatherParams{Since: "1h"},
			shouldContain: []string{"name: MUST_GATHER_SINCE", "value: 1h"},
		},
		{
			name:      "returns error for invalid since format",
			params:    PlanMustGatherParams{Since: "not-a-duration"},
			wantError: "since duration is not valid",
		},
		{
			name:          "generates plan with custom gather command",
			params:        PlanMustGatherParams{GatherCommand: "/custom/gather/script"},
			shouldContain: []string{"/custom/gather/script"},
		},
		{
			name:          "generates plan with node selector",
			params:        PlanMustGatherParams{NodeSelector: map[string]string{"node-role.kubernetes.io/worker": ""}},
			shouldContain: []string{"nodeSelector:", "node-role.kubernetes.io/worker"},
		},
		{
			name:          "generates plan with cleanup instructions when keep_resources is false",
			params:        PlanMustGatherParams{KeepResources: false},
			shouldContain: []string{"cleanup the created resources"},
		},
		{
			name:             "generates plan without cleanup instructions when keep_resources is true",
			params:           PlanMustGatherParams{KeepResources: true},
			shouldNotContain: []string{"cleanup the created resources"},
		},
		{
			name:          "cleans source dir path",
			params:        PlanMustGatherParams{SourceDir: "/custom/path/../gather/./dir"},
			shouldContain: []string{"mountPath: /custom/gather/dir"},
		},
		{
			name:          "generates plan with timeout and gather command combined",
			params:        PlanMustGatherParams{Timeout: "15m", GatherCommand: "/custom/gather"},
			shouldContain: []string{"/usr/bin/timeout 15m /custom/gather"},
		},
		{
			name: "generates plan with all parameters combined",
			params: PlanMustGatherParams{
				Namespace:    "test-ns",
				NodeName:     "node-1",
				HostNetwork:  true,
				SourceDir:    "/gather-output",
				Since:        "2h",
				Timeout:      "45m",
				Images:       []string{"quay.io/test/gather:v1"},
				NodeSelector: map[string]string{"kubernetes.io/os": "linux"},
			},
			shouldContain: []string{
				"namespace: test-ns",
				"nodeName: node-1",
				"hostNetwork: true",
				"mountPath: /gather-output",
				"value: 2h",
				"/usr/bin/timeout 45m",
				"image: quay.io/test/gather:v1",
				"kubernetes.io/os",
			},
		},
		{
			name:             "handles empty string timeout",
			params:           PlanMustGatherParams{Timeout: ""},
			shouldContain:    []string{"/usr/bin/gather"},
			shouldNotContain: []string{"/usr/bin/timeout"},
		},
		{
			name:             "handles empty string since",
			params:           PlanMustGatherParams{Since: ""},
			shouldNotContain: []string{"MUST_GATHER_SINCE"},
		},
		{
			name:          "handles empty images slice",
			params:        PlanMustGatherParams{Images: []string{}},
			shouldContain: []string{"image: registry.redhat.io/openshift4/ose-must-gather:latest"},
		},
		{
			name:   "handles nil node selector",
			params: PlanMustGatherParams{NodeSelector: nil},
		},
		{
			name:   "handles empty node selector map",
			params: PlanMustGatherParams{NodeSelector: map[string]string{}},
		},
		{
			name:          "includes wait container in pod spec",
			params:        PlanMustGatherParams{},
			shouldContain: []string{"name: wait", "sleep infinity"},
		},
		{
			name:          "includes tolerations for all taints",
			params:        PlanMustGatherParams{},
			shouldContain: []string{"tolerations:", "operator: Exists"},
		},
		{
			name:          "includes priority class",
			params:        PlanMustGatherParams{},
			shouldContain: []string{"priorityClassName: system-cluster-critical"},
		},
		{
			name:          "includes restart policy never",
			params:        PlanMustGatherParams{},
			shouldContain: []string{"restartPolicy: Never"},
		},
		{
			name:          "includes cluster-admin role binding",
			params:        PlanMustGatherParams{},
			shouldContain: []string{"name: cluster-admin", "kind: ClusterRole"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeclient.NewFakeKubernetesClient()

			result, err := PlanMustGather(ctx, client, tt.params)

			if tt.wantError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantError)
				require.Empty(t, result)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, result)

			for _, want := range tt.shouldContain {
				require.Contains(t, result, want)
			}
			for _, notWant := range tt.shouldNotContain {
				require.NotContains(t, result, notWant)
			}
		})
	}

	sarcTests := []struct {
		name          string
		permissions   []fakeclient.Option
		params        PlanMustGatherParams
		shouldContain string
	}{
		{
			name: "includes warning when no namespace create permission",
			permissions: []fakeclient.Option{
				fakeclient.WithDeniedAccess("create", "", "namespaces", "", ""),
			},
			shouldContain: "WARNING: The resources_create_or_update call does not have permission to create namespace(s)",
		},
		{
			name: "includes warning when no serviceaccount create permission",
			permissions: []fakeclient.Option{
				fakeclient.WithDeniedAccess("create", "", "serviceaccounts", "", ""),
			},
			shouldContain: "WARNING: The resources_create_or_update call does not have permission to create serviceaccount(s)",
		},
		{
			name: "includes warning when no clusterrolebinding create permission",
			permissions: []fakeclient.Option{
				fakeclient.WithDeniedAccess("create", "rbac.authorization.k8s.io", "clusterrolebindings", "", ""),
			},
			shouldContain: "WARNING: The resources_create_or_update call does not have permission to create clusterrolebinding(s)",
		},
		{
			name: "includes warning when no pod create permission",
			permissions: []fakeclient.Option{
				fakeclient.WithDeniedAccess("create", "", "pods", "", ""),
			},
			shouldContain: "WARNING: The resources_create_or_update call does not have permission to create pod(s)",
		},
		{
			name: "includes warning when hostNetwork enabled without SCC permission",
			permissions: []fakeclient.Option{
				fakeclient.WithDeniedAccess("use", "security.openshift.io", "securitycontextconstraints", "", ""),
			},
			shouldContain: "WARNING: The resources_create_or_update call does not have permission to create pod(s) with hostNetwork: true",
			params: PlanMustGatherParams{
				HostNetwork: true,
			},
		},
	}

	for _, tt := range sarcTests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeclient.NewFakeKubernetesClient(tt.permissions...)

			result, err := PlanMustGather(ctx, client, tt.params)

			require.NoError(t, err)
			require.NotEmpty(t, result)
			require.Contains(t, result, tt.shouldContain)
		})
	}
}
