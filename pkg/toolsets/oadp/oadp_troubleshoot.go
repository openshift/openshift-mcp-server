package oadp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	coretoolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/core"
)

// initOADPTroubleshoot initializes the OADP troubleshooting prompt
func initOADPTroubleshoot() []api.ServerPrompt {
	return []api.ServerPrompt{
		{
			Prompt: config.Prompt{
				Name:        "oadp-troubleshoot",
				Title:       "OADP Troubleshoot",
				Description: "Generate a step-by-step troubleshooting guide for diagnosing OADP backup and restore issues." + coretoolset.ListPageSentence,
				Arguments: []config.PromptArgument{
					{
						Name:        "namespace",
						Description: "The OADP namespace",
						Required:    true,
					},
					{
						Name:        "backup",
						Description: "The name of a specific backup to troubleshoot",
						Required:    false,
					},
					{
						Name:        "restore",
						Description: "The name of a specific restore to troubleshoot",
						Required:    false,
					},
					{
						Name:        "limit",
						Description: coretoolset.LimitDescription + " Applied to each Kubernetes list in this prompt.",
						Required:    false,
					},
					{
						Name:        "continue",
						Description: coretoolset.ContinueDescription + " For this prompt the token pages Warning events (`# list: events`). Continue tokens printed for other lists belong to resources_list.",
						Required:    false,
					},
				},
			},
			RBAC: api.RBACBounded(
				api.RBACRequirement{
					Verbs: []string{"list"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "oadp.openshift.io",
						Resource: "dataprotectionapplications",
					}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs: []string{"list"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "velero.io",
						Resource: "backupstoragelocations",
					}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs: []string{"list"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "velero.io",
						Resource: "backups",
					}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs: []string{"get"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "velero.io",
						Resource: "backups",
					}},
					Namespace:    &api.RBACNamespace{Argument: "namespace"},
					ResourceName: &api.RBACResourceName{Argument: "backup"},
				},
				api.RBACRequirement{
					Verbs: []string{"list"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "velero.io",
						Resource: "restores",
					}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs: []string{"get"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						APIGroup: "velero.io",
						Resource: "restores",
					}},
					Namespace:    &api.RBACNamespace{Argument: "namespace"},
					ResourceName: &api.RBACResourceName{Argument: "restore"},
				},
				api.RBACRequirement{
					Verbs:     []string{"list"},
					Target:    api.RBACTarget{Resource: &api.RBACResourceTarget{Resource: "pods"}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs: []string{"get"},
					Target: api.RBACTarget{Resource: &api.RBACResourceTarget{
						Resource:    "pods",
						Subresource: "log",
					}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
				api.RBACRequirement{
					Verbs:     []string{"list"},
					Target:    api.RBACTarget{Resource: &api.RBACResourceTarget{Resource: "events"}},
					Namespace: &api.RBACNamespace{Argument: "namespace"},
				},
			),
			Handler: oadpTroubleshootHandler,
		},
	}
}

// oadpTroubleshootHandler implements the OADP troubleshooting prompt
func oadpTroubleshootHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	args := params.GetArguments()
	namespace := args["namespace"]
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	backupName := args["backup"]
	restoreName := args["restore"]
	page, err := promptListPage(args)
	if err != nil {
		return nil, err
	}

	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	dynamicClient := params.DynamicClient()

	// Each section is fetched only after its preceding text fits. The nested
	// budget also bounds item loops within each section.
	guideTemplate := `# OADP Troubleshooting Guide

## Namespace: ` + namespace + `

Use this guide to diagnose issues with OADP backups and restores. The relevant resource data has been collected below.

---

## Step 1: DataProtectionApplication (DPA) Status

The DPA is the central configuration for OADP. Check:
- Does a DPA exist?
- Are the conditions healthy (type "Reconciled" with status "True")?
- Is the backup storage location configured correctly?

%s

---

## Step 2: BackupStorageLocation (BSL) Status

BSLs define where backups are stored. Check:
- phase should be "Available"
- If "Unavailable": credentials may be wrong, bucket may not exist, or network access may be blocked
- lastValidationTime should be recent

%s

---

## Step 3: Backups

Review backup status:
- phase "Completed" = success
- phase "PartiallyFailed" = some resources failed to back up (check errors/warnings)
- phase "Failed" = backup failed entirely
- phase "InProgress" for extended time = backup may be stuck

%s

---

## Step 4: Restores

Review restore status:
- phase "Completed" = success
- phase "PartiallyFailed" = some resources failed to restore
- phase "Failed" = restore failed entirely
- Check that the referenced backup exists and is in "Completed" phase

%s

---

## Step 5: Velero Pod Health

Check the Velero server pod:
- Pod should be "Running" with all containers ready
- Check for restart counts (frequent restarts indicate crashes)
- nodeAgent pods should also be running if file-system backup is configured

%s

---

## Step 6: Velero Pod Logs

Review logs from the Velero server for errors:
- Look for "error" or "level=error" messages
- Check for credential/authentication failures
- Check for storage access issues

%s

---

## Step 7: Events

Review events in the OADP namespace:
- Warning events indicate problems
- Look for scheduling, storage, or permission issues

%s

---

## Troubleshooting Analysis

Based on the data above, analyze:

1. **DPA Health**: Is the DPA reconciled and healthy?
2. **BSL Availability**: Is the backup storage location "Available"?
3. **Backup/Restore Status**: Are operations completing successfully or failing?
4. **Velero Pod**: Is the Velero server running without restarts?
5. **Credentials**: Are cloud credentials configured correctly?
6. **Events**: Are there Warning events indicating problems?

---

## Common Issues and Fixes

1. **BSL "Unavailable"**: Check cloud credentials secret exists and has correct keys. Verify bucket exists and is accessible.
2. **Backup stuck "InProgress"**: Check Velero pod logs for errors. The pod may need restarting.
3. **Backup "PartiallyFailed"**: Some resources could not be backed up. Check backup status for per-resource errors.
4. **Restore "Failed"**: The referenced backup may not exist or may not be in "Completed" phase. Check that the BSL containing the backup is available.
5. **DPA not reconciled**: The OADP operator may not be running. Check the operator pod in the openshift-adp namespace.
6. **NodeAgent not running**: If using file-system backups, ensure nodeAgent is enabled in the DPA configuration.

Use the OADP tools to apply fixes (create/update DPA, check storage locations, retry backups).

---

## Report Findings

After completing troubleshooting and attempting fixes, report:
- **Status:** Healthy/Degraded/Failed
- **Root Cause:** Description or "None found"
- **Action Taken:** What was done to fix the issue (or "None" if no fix was needed/possible)
- **Result:** Whether the fix was successful or further action is needed
`
	parts := strings.Split(guideTemplate, "%s")
	guide := api.NewPromptBudget(ctx, "I'll analyze the collected OADP data to diagnose backup and restore issues systematically.")
	var podNames []string
	fetches := []func(context.Context) string{
		func(ctx context.Context) string { return fetchDPAStatus(ctx, dynamicClient, namespace, page) },
		func(ctx context.Context) string { return fetchBSLStatus(ctx, dynamicClient, namespace, page) },
		func(ctx context.Context) string {
			if backupName != "" {
				return fetchBackupDetails(ctx, dynamicClient, namespace, backupName)
			}
			return fetchRecentBackups(ctx, dynamicClient, namespace, page)
		},
		func(ctx context.Context) string {
			if restoreName != "" {
				return fetchRestoreDetails(ctx, dynamicClient, namespace, restoreName)
			}
			return fetchRecentRestores(ctx, dynamicClient, namespace, page)
		},
		func(ctx context.Context) string {
			text, names := fetchVeleroPods(ctx, dynamicClient, namespace, page)
			podNames = names
			return text
		},
		func(ctx context.Context) string {
			return fetchVeleroPodLogs(ctx, params.KubernetesClient, namespace, podNames)
		},
		func(ctx context.Context) string {
			return fetchOADPEvents(ctx, params.KubernetesClient, namespace, page)
		},
	}
	steps := make([]api.Step, 0, len(parts)+len(fetches))
	for i, fetch := range fetches {
		steps = append(steps, api.Text(parts[i]))
		steps = append(steps, func(b *api.BuilderWithBudget) error {
			// A nested collector may have its own truncation notice. Commit
			// only its prefix so the parent appends the notice once.
			text := fetch(api.WithResultBudget(ctx, b.Remaining()))
			returnCut := strings.HasSuffix(text, api.ResultTruncationNotice)
			b.WriteCut(strings.TrimSuffix(text, api.ResultTruncationNotice), returnCut)
			return nil
		})
	}
	steps = append(steps, api.Text(parts[len(parts)-1]))
	_ = guide.Sections(steps...)

	return api.NewPromptCallResult(
		"OADP troubleshooting guide generated",
		[]api.PromptMessage{
			{
				Role: "user",
				Content: api.PromptContent{
					Type: "text",
					Text: guide.String(),
				},
			},
			{
				Role: "assistant",
				Content: api.PromptContent{
					Type: "text",
					Text: "I'll analyze the collected OADP data to diagnose backup and restore issues systematically.",
				},
			},
		},
		nil,
	), nil
}

// fetchDPAStatus fetches all DPA resources and returns their status as YAML
func fetchDPAStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace string, page metav1.ListOptions) string {
	dpas, err := oadp.ListDataProtectionApplications(ctx, dynamicClient, namespace, metav1.ListOptions{Limit: page.Limit})
	if err != nil {
		return fmt.Sprintf("### DataProtectionApplication\n\n*Error listing DPAs: %v*", coretoolset.ListErr(err))
	}

	if len(dpas.Items) == 0 {
		return "### DataProtectionApplication\n\n*No DataProtectionApplication found — OADP may not be configured*"
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString("### DataProtectionApplication\n\n")
	steps := make([]api.Step, 0, len(dpas.Items))
	for _, dpa := range dpas.Items {
		steps = append(steps, func(result *api.BuilderWithBudget) error {
			status, found, err := unstructured.NestedMap(dpa.Object, "status")
			if err != nil {
				result.WriteString(fmt.Sprintf("#### %s\n\n*Error extracting status: %v*\n\n", dpa.GetName(), err))
				return nil
			}
			if !found {
				result.WriteString(fmt.Sprintf("#### %s\n\n*No status found (DPA may still be reconciling)*\n\n", dpa.GetName()))
				return nil
			}
			yamlStr, err := output.MarshalYaml(status)
			if err != nil {
				result.WriteString(fmt.Sprintf("#### %s\n\n*Error marshaling status: %v*\n\n", dpa.GetName(), err))
				return nil
			}
			result.WriteString(fmt.Sprintf("#### %s\n\n```yaml\n%s```\n\n", dpa.GetName(), yamlStr))
			return nil
		})
	}
	_ = result.Sections(steps...)

	return prependPageNote(result.String(), "dataprotectionapplications", dpas)
}

// fetchBSLStatus fetches all BSL resources and returns their status as YAML
func fetchBSLStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace string, page metav1.ListOptions) string {
	bsls, err := oadp.ListBackupStorageLocations(ctx, dynamicClient, namespace, metav1.ListOptions{Limit: page.Limit})
	if err != nil {
		return fmt.Sprintf("### BackupStorageLocations\n\n*Error listing BSLs: %v*", coretoolset.ListErr(err))
	}

	if len(bsls.Items) == 0 {
		return "### BackupStorageLocations\n\n*No BackupStorageLocations found*"
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString("### BackupStorageLocations\n\n")
	steps := make([]api.Step, 0, len(bsls.Items))
	for _, bsl := range bsls.Items {
		steps = append(steps, func(result *api.BuilderWithBudget) error {
			phase, _, err := unstructured.NestedString(bsl.Object, "status", "phase")
			if err != nil {
				phase = ""
			}
			lastValidation, _, err := unstructured.NestedString(bsl.Object, "status", "lastValidationTime")
			if err != nil {
				lastValidation = ""
			}
			provider, _, err := unstructured.NestedString(bsl.Object, "spec", "provider")
			if err != nil {
				provider = ""
			}
			bucket, _, err := unstructured.NestedString(bsl.Object, "spec", "objectStorage", "bucket")
			if err != nil {
				bucket = ""
			}

			result.WriteString(fmt.Sprintf("#### %s\n\n- **Phase:** %s\n- **Provider:** %s\n- **Bucket:** %s\n- **Last Validation:** %s\n\n",
				bsl.GetName(), valueOrNA(phase), valueOrNA(provider), valueOrNA(bucket), valueOrNA(lastValidation)))
			return nil
		})
	}
	_ = result.Sections(steps...)

	return prependPageNote(result.String(), "backupstoragelocations", bsls)
}

// fetchRecentBackups fetches the most recent backups and summarizes their status
func fetchRecentBackups(ctx context.Context, dynamicClient dynamic.Interface, namespace string, page metav1.ListOptions) string {
	backups, err := oadp.ListBackups(ctx, dynamicClient, namespace, metav1.ListOptions{Limit: page.Limit})
	if err != nil {
		return fmt.Sprintf("### Recent Backups\n\n*Error listing backups: %v*", coretoolset.ListErr(err))
	}

	if len(backups.Items) == 0 {
		return "### Recent Backups\n\n*No backups found*"
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString("### Recent Backups\n\n")
	result.WriteString("| Name | Phase | Started | Completed | Errors | Warnings |\n")
	result.WriteString("|------|-------|---------|-----------|--------|----------|\n")

	steps := make([]api.Step, 0, len(backups.Items))
	for _, backup := range backups.Items {
		steps = append(steps, func(result *api.BuilderWithBudget) error {
			phase, _, err := unstructured.NestedString(backup.Object, "status", "phase")
			if err != nil {
				phase = ""
			}
			startTime, _, err := unstructured.NestedString(backup.Object, "status", "startTimestamp")
			if err != nil {
				startTime = ""
			}
			completionTime, _, err := unstructured.NestedString(backup.Object, "status", "completionTimestamp")
			if err != nil {
				completionTime = ""
			}
			errors, _, err := unstructured.NestedInt64(backup.Object, "status", "errors")
			if err != nil {
				errors = 0
			}
			warnings, _, err := unstructured.NestedInt64(backup.Object, "status", "warnings")
			if err != nil {
				warnings = 0
			}

			result.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %d |\n",
				backup.GetName(), valueOrNA(phase), valueOrNA(startTime), valueOrNA(completionTime), errors, warnings))
			return nil
		})
	}
	_ = result.Sections(steps...)
	result.WriteString("\n")

	return prependPageNote(result.String(), "backups", backups)
}

// fetchBackupDetails fetches a specific backup and returns full details
func fetchBackupDetails(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) string {
	backup, err := oadp.GetBackup(ctx, dynamicClient, namespace, name)
	if err != nil {
		return fmt.Sprintf("### Backup: %s\n\n*Error fetching backup: %v*", name, err)
	}

	status, found, err := unstructured.NestedMap(backup.Object, "status")
	if err != nil {
		return fmt.Sprintf("### Backup: %s\n\n*Error extracting status: %v*", name, err)
	}
	if !found {
		return fmt.Sprintf("### Backup: %s\n\n*No status found (backup may not have started)*", name)
	}

	spec, _, err := unstructured.NestedMap(backup.Object, "spec")
	if err != nil {
		spec = nil
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString(fmt.Sprintf("### Backup: %s\n\n", name))

	if spec != nil {
		specYaml, err := output.MarshalYaml(spec)
		if err == nil {
			result.WriteString(fmt.Sprintf("**Spec:**\n```yaml\n%s```\n\n", specYaml))
		}
	}

	if result.Fits(0) {
		statusYaml, err := output.MarshalYaml(status)
		if err == nil {
			result.WriteString(fmt.Sprintf("**Status:**\n```yaml\n%s```\n\n", statusYaml))
		}
	}

	return result.String()
}

// fetchRecentRestores fetches the most recent restores and summarizes their status
func fetchRecentRestores(ctx context.Context, dynamicClient dynamic.Interface, namespace string, page metav1.ListOptions) string {
	restores, err := oadp.ListRestores(ctx, dynamicClient, namespace, metav1.ListOptions{Limit: page.Limit})
	if err != nil {
		return fmt.Sprintf("### Recent Restores\n\n*Error listing restores: %v*", coretoolset.ListErr(err))
	}

	if len(restores.Items) == 0 {
		return "### Recent Restores\n\n*No restores found*"
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString("### Recent Restores\n\n")
	result.WriteString("| Name | Phase | Backup | Started | Completed | Errors | Warnings |\n")
	result.WriteString("|------|-------|--------|---------|-----------|--------|----------|\n")

	steps := make([]api.Step, 0, len(restores.Items))
	for _, restore := range restores.Items {
		steps = append(steps, func(result *api.BuilderWithBudget) error {
			phase, _, err := unstructured.NestedString(restore.Object, "status", "phase")
			if err != nil {
				phase = ""
			}
			backupName, _, err := unstructured.NestedString(restore.Object, "spec", "backupName")
			if err != nil {
				backupName = ""
			}
			startTime, _, err := unstructured.NestedString(restore.Object, "status", "startTimestamp")
			if err != nil {
				startTime = ""
			}
			completionTime, _, err := unstructured.NestedString(restore.Object, "status", "completionTimestamp")
			if err != nil {
				completionTime = ""
			}
			errors, _, err := unstructured.NestedInt64(restore.Object, "status", "errors")
			if err != nil {
				errors = 0
			}
			warnings, _, err := unstructured.NestedInt64(restore.Object, "status", "warnings")
			if err != nil {
				warnings = 0
			}

			result.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d | %d |\n",
				restore.GetName(), valueOrNA(phase), valueOrNA(backupName), valueOrNA(startTime), valueOrNA(completionTime), errors, warnings))
			return nil
		})
	}
	_ = result.Sections(steps...)
	result.WriteString("\n")

	return prependPageNote(result.String(), "restores", restores)
}

// fetchRestoreDetails fetches a specific restore and returns full details
func fetchRestoreDetails(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) string {
	restore, err := oadp.GetRestore(ctx, dynamicClient, namespace, name)
	if err != nil {
		return fmt.Sprintf("### Restore: %s\n\n*Error fetching restore: %v*", name, err)
	}

	status, found, err := unstructured.NestedMap(restore.Object, "status")
	if err != nil {
		return fmt.Sprintf("### Restore: %s\n\n*Error extracting status: %v*", name, err)
	}
	if !found {
		return fmt.Sprintf("### Restore: %s\n\n*No status found (restore may not have started)*", name)
	}

	spec, _, err := unstructured.NestedMap(restore.Object, "spec")
	if err != nil {
		spec = nil
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString(fmt.Sprintf("### Restore: %s\n\n", name))

	if spec != nil {
		specYaml, err := output.MarshalYaml(spec)
		if err == nil {
			result.WriteString(fmt.Sprintf("**Spec:**\n```yaml\n%s```\n\n", specYaml))
		}
	}

	if result.Fits(0) {
		statusYaml, err := output.MarshalYaml(status)
		if err == nil {
			result.WriteString(fmt.Sprintf("**Status:**\n```yaml\n%s```\n\n", statusYaml))
		}
	}

	return result.String()
}

// fetchVeleroPods fetches the Velero server and NodeAgent pods
func fetchVeleroPods(ctx context.Context, dynamicClient dynamic.Interface, namespace string, page metav1.ListOptions) (string, []string) {
	gvr := podGVR()

	// Fetch Velero server pods
	podOpts := metav1.ListOptions{Limit: page.Limit, LabelSelector: "app.kubernetes.io/name=velero"}
	podList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, podOpts)
	if err != nil {
		return fmt.Sprintf("### Velero Pods\n\n*Error listing pods: %v*", coretoolset.ListErr(err)), nil
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	var podNames []string
	result.WriteString("### Velero Pods\n\n")
	result.WriteString("| Name | Phase | Ready | Restarts | Age |\n")
	result.WriteString("|------|-------|-------|----------|-----|\n")

	appendPods := func(pods []unstructured.Unstructured) {
		steps := make([]api.Step, 0, len(pods))
		for _, pod := range pods {
			steps = append(steps, func(result *api.BuilderWithBudget) error {
				phase, _, err := unstructured.NestedString(pod.Object, "status", "phase")
				if err != nil {
					phase = ""
				}
				creationTime := pod.GetCreationTimestamp().Format("2006-01-02T15:04:05Z")

				// Get ready and restart counts from container statuses
				ready, restarts := getPodReadyAndRestarts(&pod)

				result.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s |\n",
					pod.GetName(), phase, ready, restarts, creationTime))
				if !result.Truncated() {
					podNames = append(podNames, pod.GetName())
				}
				return nil
			})
		}
		_ = result.Sections(steps...)
	}
	appendPods(podList.Items)
	var nodeAgentList *unstructured.UnstructuredList
	if result.Fits(0) {
		// Fetch node agents only if the section still has space.
		nodeOpts := metav1.ListOptions{Limit: page.Limit, LabelSelector: "name=node-agent"}
		nodeAgentList, err = dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, nodeOpts)
		if err == nil {
			appendPods(nodeAgentList.Items)
		}
	}
	var text string
	if len(podList.Items) == 0 && len(podNames) == 0 && !result.Truncated() {
		text = "### Velero Pods\n\n*No Velero pods found — the OADP operator may not be installed or the DPA may not be configured*"
	} else {
		result.WriteString("\n")
		text = result.String()
	}
	text = prependPageNote(text, "pods", podList)
	if nodeAgentList != nil {
		text = prependPageNote(text, "node-agent pods", nodeAgentList)
	}
	return text, podNames
}

// fetchVeleroPodLogs fetches logs from the Velero server pod
func fetchVeleroPodLogs(ctx context.Context, client api.KubernetesClient, namespace string, podNames []string) string {
	if len(podNames) == 0 {
		return "### Velero Pod Logs\n\n*No Velero pods found — no logs available*"
	}

	core := kubernetes.NewCore(client)
	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	result.WriteString("### Velero Pod Logs\n\n")

	steps := make([]api.Step, 0, len(podNames))
	for _, podName := range podNames {
		// Only fetch logs from velero server pods, not node-agent
		if strings.HasPrefix(podName, "node-agent") {
			continue
		}
		steps = append(steps, func(result *api.BuilderWithBudget) error {
			result.WriteString(fmt.Sprintf("#### Pod: %s\n\n", podName))
			if result.Truncated() {
				return nil
			}
			logs, err := core.PodsLog(ctx, namespace, podName, "velero", false, 100)
			if err != nil {
				result.WriteString(fmt.Sprintf("*Error fetching logs: %v*\n\n", err))
				return nil
			}

			// Filter for error/warning lines to keep output focused
			lines := result.Scratch(len("**Error/Warning lines (last 100 log lines):**\n```\n\n```\n\n"))
			filterLogLines(logs, []string{"error", "level=error", "level=warning", "Failed", "forbidden"}, lines)
			if strings.TrimSpace(lines.String()) != "" {
				result.CommitScratch("**Error/Warning lines (last 100 log lines):**\n```\n", "\n```\n\n", lines)
			} else if lines.Truncated() {
				result.WriteString("**Error/Warning lines (last 100 log lines):**\n```")
				result.WriteString("\n")
			} else {
				result.WriteString("*No error or warning lines found in recent logs*\n\n")
			}
			return nil
		})
	}
	_ = result.Sections(steps...)

	return result.String()
}

// fetchOADPEvents fetches events in the OADP namespace
func fetchOADPEvents(ctx context.Context, client api.KubernetesClient, namespace string, page metav1.ListOptions) string {
	page.FieldSelector = "type=Warning"
	kcore := kubernetes.NewCore(client)
	eventMap, meta, err := kcore.EventsList(ctx, namespace, api.ListOptions{ListOptions: page})
	if err != nil {
		return fmt.Sprintf("### Events\n\n*Error listing events: %v*", coretoolset.ListErr(err))
	}

	if len(eventMap) == 0 {
		return prependPageNote("### Events\n\n*No Warning events found — this is good*", "events", meta)
	}

	// Filter for warning events which indicate problems
	result := api.NewBuilderWithBudget(api.ResultLimit(ctx))
	body := result.Scratch(len("### Warning Events\n\n```yaml\n```"))
	steps := make([]api.Step, 0, len(eventMap))
	for _, event := range eventMap {
		eventType, _ := event["Type"].(string)
		if eventType == "Warning" {
			steps = append(steps, func(body *api.BuilderWithBudget) error {
				yamlStr, err := output.MarshalYaml(event)
				if err != nil {
					body.WriteString(fmt.Sprintf("*Error marshaling event: %v*\n", err))
					return nil
				}
				body.WriteString(yamlStr)
				return nil
			})
		}
	}

	if len(steps) == 0 {
		return prependPageNote("### Events\n\n*No Warning events found — this is good*", "events", meta)
	}
	if body.Truncated() {
		result.WriteString("### Warning Events\n\n```yaml\n```")
		result.WriteString("\n")
		return prependPageNote(result.String(), "events", meta)
	}
	_ = body.Sections(steps...)
	result.CommitScratch("### Warning Events\n\n```yaml\n", "```", body)
	return prependPageNote(result.String(), "events", meta)
}

// promptListPage reads the prompt's string arguments. ApplyListPage only accepts
// a JSON number, which prompt arguments are not.
func promptListPage(args map[string]string) (metav1.ListOptions, error) {
	opts := metav1.ListOptions{Continue: args["continue"]}
	raw := strings.TrimSpace(args["limit"])
	if raw == "" {
		return opts, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return metav1.ListOptions{}, fmt.Errorf("limit parameter must be an integer")
	}
	if limit < 0 {
		return metav1.ListOptions{}, fmt.Errorf("limit must be greater than or equal to 0")
	}
	opts.Limit = limit
	return opts, nil
}

func prependPageNote(text, label string, listed any) string {
	note, _ := coretoolset.WithListPage("", nil, listed)
	if note == "" {
		return text
	}
	if label != "" {
		note = "# list: " + label + "\n" + note
	}
	return note + text
}

// Helper functions

func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func getPodReadyAndRestarts(pod *unstructured.Unstructured) (string, int64) {
	containerStatuses, found, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	if !found {
		return "N/A", 0
	}

	totalContainers := len(containerStatuses)
	readyCount := 0
	var totalRestarts int64

	for _, cs := range containerStatuses {
		csMap, ok := cs.(map[string]any)
		if !ok {
			continue
		}
		if ready, ok := csMap["ready"].(bool); ok && ready {
			readyCount++
		}
		switch restarts := csMap["restartCount"].(type) {
		case int64:
			totalRestarts += restarts
		case float64:
			totalRestarts += int64(restarts)
		}
	}

	return fmt.Sprintf("%d/%d", readyCount, totalContainers), totalRestarts
}

func filterLogLines(logs string, keywords []string, result *api.BuilderWithBudget) {
	steps := make([]api.Step, 0)
	for line := range strings.SplitSeq(logs, "\n") {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				steps = append(steps, api.Text(line+"\n"))
				break
			}
		}
	}
	_ = result.Sections(steps...)
}

// podGVR returns the Pod GVR
func podGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
}
