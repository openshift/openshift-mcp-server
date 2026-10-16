package processor

import (
	"regexp"

	"github.com/openshift/cluster-health-analyzer/pkg/common"
	"github.com/prometheus/common/model"
)

// This file contains data used to map the signal to particular components.

var (
	nodeAlerts []model.LabelValue = []model.LabelValue{
		"NodeClockNotSynchronising",
		"KubeNodeNotReady",
		"KubeNodeUnreachable",
		"NodeSystemSaturation",
		"NodeFilesystemSpaceFillingUp",
		"NodeFilesystemAlmostOutOfSpace",
		"NodeMemoryMajorPagesFaults",
		"NodeNetworkTransmitErrs",
		"NodeTextFileCollectorScrapeError",
		"NodeFilesystemFilesFillingUp",
		"NodeNetworkReceiveErrs",
		"NodeClockSkewDetected",
		"NodeFilesystemAlmostOutOfFiles",
		"NodeWithoutOVNKubeNodePodRunning",
		"InfraNodesNeedResizingSRE",
		"NodeHighNumberConntrackEntriesUsed",
		"NodeMemHigh",
		"NodeNetworkInterfaceFlapping",
		"NodeWithoutSDNPod",
		"NodeCpuHigh",
		"CriticalNodeNotReady",
		"NodeFileDescriptorLimit",
		// subset of MCO alerts https://github.com/openshift/machine-config-operator/blob/204767253e30608b5b7fd70ad1ace02ba1d64b46/install/0000_90_machine-config_01_prometheus-rules.yaml#L115
		"MCCPoolAlert",
		"MCCDrainError",
		"MCDRebootError",
		"MCDPivotError",
	}

	coreMatchers = []componentMatcher{
		{"etcd", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace",
				common.NewStringValuesMatcher(
					"openshift-etcd",
					"openshift-etcd-operator"))}},
		{"kube-apiserver", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace",
				common.NewStringValuesMatcher(
					"openshift-kube-apiserver",
					"openshift-kube-apiserver-operator"))}},
		{"kube-controller-manager", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-kube-controller-manager",
				"openshift-kube-controller-manager-operator",
				"kube-system",
			))}},
		{"kube-scheduler", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-kube-scheduler",
				"openshift-kube-scheduler-operator",
			))}},
		{"machine-approver", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cluster-machine-approver",
				"openshift-machine-approver-operator",
			))}},
		{"machine-config", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-machine-config-operator",
			)),
			common.NewLabelsMatcher("alertname", common.NewStringValuesMatcher(
				"HighOverallControlPlaneMemory",
				"ExtremelyHighIndividualControlPlaneMemory",
				"MissingMachineConfig",
				"MCCBootImageUpdateError",
				"KubeletHealthState",
				"SystemMemoryExceedsReservation",
			))}},
		{"version", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cluster-version",
				"openshift-version-operator",
			)),
			common.NewLabelsMatcher("alertname", common.NewStringValuesMatcher(
				"ClusterNotUpgradeable",
				"UpdateAvailable",
			))}},
		{"dns", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-dns",
				"openshift-dns-operator",
			))}},
		{"authentication", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-authentication",
				"openshift-oauth-apiserver",
				"openshift-authentication-operator",
			))}},
		{"cert-manager", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cert-manager",
				"openshift-cert-manager-operator",
			))}},
		{"cloud-controller-manager", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cloud-controller-manager",
				"openshift-cloud-controller-manager-operator",
			))}},
		{"cloud-credential", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cloud-credential-operator",
			))}},
		{"cluster-api", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cluster-api",
				"openshift-cluster-api-operator",
			))}},
		{"config-operator", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-config-operator",
			))}},
		{"kube-storage-version-migrator", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-kube-storage-version-migrator",
				"openshift-kube-storage-version-migrator-operator",
			))}},
		{"image-registry", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-image-registry",
				"openshift-image-registry-operator",
			))}},
		{"ingress", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-ingress",
				"openshift-route-controller-manager",
				"openshift-ingress-canary",
				"openshift-ingress-operator",
			))}},
		{"console", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-console",
				"openshift-console-operator",
			))}},
		{"insights", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-insights",
				"openshift-insights-operator",
			))}},
		{"machine-api", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-machine-api",
				"openshift-machine-api-operator",
			))}},
		{"monitoring", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-monitoring",
				"openshift-monitoring-operator",
			))}},
		{"network", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-network-operator",
				"openshift-ovn-kubernetes",
				"openshift-multus",
				"openshift-network-diagnostics",
				"openshift-sdn",
			))}},
		{"node-tuning", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cluster-node-tuning-operator",
				"openshift-node-tuning-operator",
			))}},
		{"openshift-apiserver", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-apiserver",
				"openshift-apiserver-operator",
			))}},
		{"openshift-controller-manager", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-controller-manager",
				"openshift-controller-manager-operator",
			))}},
		{"openshift-samples", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cluster-samples-operator",
				"openshift-samples-operator",
			))}},
		{"operator-lifecycle-manager", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-operator-lifecycle-manager",
			))}},
		{"service-ca", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-service-ca",
				"openshift-service-ca-operator",
			))}},
		{"storage", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-storage",
				"openshift-cluster-csi-drivers",
				"openshift-cluster-storage-operator",
				"openshift-storage-operator",
			))}},
		{"vertical-pod-autoscaler", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-vertical-pod-autoscaler",
				"openshift-vertical-pod-autoscaler-operator",
			))}},
		{"marketplace", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-marketplace",
				"openshift-marketplace-operator",
			)),
		},
		},
	}

	workloadMatchers = []componentMatcher{
		{"openshift-compliance", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-compliance",
			))}},
		{"openshift-file-integrity", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-file-integrity",
			))}},
		{"openshift-logging", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-logging",
			))}},
		{"openshift-user-workload-monitoring", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-user-workload-monitoring",
			))}},
		{"openshift-gitops", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-gitops",
				"openshift-gitops-operator",
			))}},
		{"openshift-operators", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-operators",
			))}},
		{"kubevirt", []common.LabelsMatcher{
			common.NewLabelsMatcher("kubernetes_operator_part_of", common.NewStringValuesMatcher(
				"kubevirt",
			)),
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-cnv",
			)),
		}},
		{"openshift-local-storage", []common.LabelsMatcher{
			common.NewLabelsMatcher("namespace", common.NewStringValuesMatcher(
				"openshift-local-storage",
			))}},
		{"quay", []common.LabelsMatcher{
			common.NewLabelsMatcher("container", common.NewStringValuesMatcher(
				"quay-app",
				"quay-mirror",
				"quay-app-upgrade",
			))}},
		{"Argo", []common.LabelsMatcher{
			common.NewLabelsMatcher("alertname", common.NewRegexValuesMatcher(
				regexp.MustCompile("^Argo"),
			))},
		},
	}
)
