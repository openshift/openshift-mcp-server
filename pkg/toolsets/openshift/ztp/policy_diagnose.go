package ztp

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

var policyGVR = schema.GroupVersionResource{
	Group:    "policy.open-cluster-management.io",
	Version:  "v1",
	Resource: "policies",
}

var placementBindingGVR = schema.GroupVersionResource{
	Group:    "policy.open-cluster-management.io",
	Version:  "v1",
	Resource: "placementbindings",
}

var argoCDAppGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

var placementDecisionGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "placementdecisions",
}

// Prompts returns the ZTP policy diagnosis prompt definitions.
func Prompts() []api.ServerPrompt {
	return []api.ServerPrompt{
		{
			Prompt: api.Prompt{
				Name:  "ztp-policy-diagnose",
				Title: "ZTP Policy Diagnosis and Remediation",
				Description: "Diagnose a NonCompliant ACM policy in a ZTP GitOps " +
					"environment. Gathers policy status, compliance history, " +
					"placement data, and ArgoCD state, then returns a multi-level " +
					"investigation workflow with pattern-matching heuristics for " +
					"common ZTP failure modes.",
				Arguments: []api.PromptArgument{
					{
						Name:        "policy_name",
						Description: "Full or partial ACM policy name (e.g. ztp-policies.web-terminal-operator-status)",
						Required:    true,
					},
					{
						Name:        "cluster",
						Description: "Target managed cluster name (e.g. sno-abi). If omitted, all clusters are analysed.",
						Required:    false,
					},
					{
						Name:        "namespace",
						Description: "Namespace where the policy lives (default: all namespaces)",
						Required:    false,
					},
				},
			},
			Handler: policyDiagnoseHandler,
		},
	}
}

func policyDiagnoseHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	args := params.GetArguments()
	policyName := args["policy_name"]
	cluster := args["cluster"]
	namespace := args["namespace"]

	if policyName == "" {
		return nil, fmt.Errorf("policy_name argument is required")
	}

	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	dynamicClient := params.DynamicClient()

	policies := findPolicies(ctx, dynamicClient, policyName, namespace)
	metadataYaml := extractPolicyMetadata(policies)
	policyYaml := renderPolicyStatus(policies)
	complianceYaml := fetchComplianceDetails(ctx, dynamicClient, policies, cluster)
	placementYaml := fetchPlacementInfo(ctx, dynamicClient, policies, namespace)

	argoRef := argoAppFromPolicies(policies)
	argoAppYaml := fetchArgoApps(ctx, dynamicClient, argoRef)

	guideText := buildInvestigationGuide(policyName, cluster, metadataYaml, policyYaml, complianceYaml, placementYaml, argoAppYaml)

	return api.NewPromptCallResult(
		"ZTP policy diagnosis workflow generated",
		[]api.PromptMessage{
			{
				Role: "user",
				Content: api.PromptContent{
					Type: "text",
					Text: guideText,
				},
			},
			{
				Role: "assistant",
				Content: api.PromptContent{
					Type: "text",
					Text: fmt.Sprintf(
						"I'll diagnose the policy **%s** following the ZTP-compliant "+
							"multi-level investigation workflow. I'll examine the policy "+
							"status, perform root cause analysis on the affected clusters, "+
							"inspect the GitOps source, and recommend a fix strategy. "+
							"I will ask for your confirmation before committing any changes.",
						policyName,
					),
				},
			},
		},
		nil,
	), nil
}

func buildInvestigationGuide(policyName, cluster, metadataYaml, policyYaml, complianceYaml, placementYaml, argoAppYaml string) string {
	clusterNote := "all managed clusters"
	if cluster != "" {
		clusterNote = fmt.Sprintf("managed cluster **%s**", cluster)
	}

	return fmt.Sprintf(`# ZTP Policy Diagnosis and Remediation

## Target: %s on %s

## Constraints

1. **source-crs/ is read-only.** Files inside any source-crs/ directory
   are shared upstream templates — NEVER modify them. All other files
   (PolicyGenerator YAMLs, custom CRs, ConfigMaps, kustomization.yaml)
   are fully editable. New files may also be created outside source-crs/.

2. **git_session_start once.** Call git_session_start once at the start
   of Level 3 for a fresh clone. Reuse the returned repo_path for all
   subsequent git operations including commit/push. Do NOT call it again.

---

## Pre-fetched Data

%s

%s

%s

%s

%s

---

## Pattern Recognition

Before deep investigation, check these indicators in the data above:

- No clusters in compliance data: Placement label mismatch (policy not
  bound to any cluster; check Placement label selectors)
- Rapid Compliant/NonCompliant alternation in history: Oscillating
  compliance (enforce mode fighting a cluster operator)
- "field is immutable" or "Invalid value" in message: Immutable field
  enforcement failure (API rejects the update)
- Empty or blank values in objectDefinition: Hub template rendering
  failure (missing ConfigMap key for fromConfigMap)
- "not found" in compliance message: Missing namespace or prerequisite
- Policy NonCompliant with status condition diff: Status condition
  mismatch (validator checks condition that never matches reality)
- Short evaluationInterval + enforce + repeated updates: Oscillation
- Resource depends on namespace from higher deploy-wave: Wave ordering
  violation (ztp-deploy-wave dependency inversion)

If a pattern matches, skip to the relevant root cause. Otherwise continue
with the full investigation.

---

## Level 1: Policy Status Analysis

The policy status, metadata, and compliance details are pre-fetched above.
Review them and identify:

- Which clusters are NonCompliant and which templates are failing
- The compliance messages (they often contain the exact failure reason)
- The remediation action (enforce vs inform) and evaluation interval
- Whether compliance history shows oscillation patterns

---

## Level 2: Root Cause Analysis

For each NonCompliant template, examine the target resource on the managed
cluster using openshift-mcp-server tools with the cluster parameter.

Determine the failure category:

- **Missing resource**: Resource does not exist on the managed cluster
- **Misconfiguration**: Resource exists but has wrong field values
- **Missing prerequisite**: A dependency (Namespace, CatalogSource,
  OperatorGroup, Subscription) is missing
- **Version/channel mismatch**: Spec references an unavailable version
  or operator channel
- **Oscillating compliance**: Enforce mode fights a cluster operator that
  continuously reverts the enforced change
- **Placement not matching**: Policy exists on hub but Placement selects
  zero clusters (label selector does not match any ManagedCluster)
- **Immutable field rejection**: API server rejects enforce because the
  field cannot be updated after creation
- **Hub template failure**: fromConfigMap resolved to empty because the
  referenced ConfigMap key does not exist
- **Wave ordering violation**: Resource depends on another resource from
  a higher ztp-deploy-wave that has not been applied yet
- **Status condition mismatch**: Validator expects a condition combination
  that the actual resource never satisfies

---

## Level 3: GitOps Source Analysis

Locate the ZTP git repository and inspect the policy source.

**Actions:**

1. Use the repoURL from the ArgoCD Application data above. If none was
   found, list ArgoCD Applications via openshift-mcp-server and extract
   the repoURL. If still unavailable, ask the user for the URL.
2. Call git_session_start once to get a fresh clone. The git-mcp-server
   has pre-configured default URL and credentials — if the repoURL
   matches the default, call git_session_start with no arguments.
   Reuse the returned repo_path for all subsequent git operations.
3. Navigate the directory structure. ZTP repositories typically have:
   - PolicyGenerator YAML files (kustomize generators)
   - source-crs/ directories with CR templates (read-only upstream)
   - ConfigMaps for hub template data
   - Profile directories for cluster configurations
4. Find the PolicyGenerator YAML that produces the failing policy.
   Examine its manifest paths, patches, placement, and remediationAction.
5. If the policy references source-crs templates, read them to understand
   their structure and defaults (but do NOT modify them).
6. Check for existing ConfigMaps, patches, or overlays that customize
   the templates.

---

## Level 4: Fix Strategy

Select the appropriate fix based on root cause. Prefer the least-invasive
approach.

**Strategy A — No Git Change Needed:**
Cluster-side operations (restart pod, approve InstallPlan, fix labels).

**Strategy B — Edit the PolicyGenerator YAML:**
Most common fix. Change patches (channel, version, field values), change
remediationAction (e.g. enforce to inform for oscillation), change
evaluationInterval, change placement labelSelector, add or remove manifest
entries, adjust ztp-deploy-wave annotations.

**Strategy C — Create or edit files outside source-crs/:**
Create new CRs, ConfigMaps, Namespace YAMLs, or PolicyGenerator files.
Any file outside source-crs/ is editable.

**Strategy D — Fix ConfigMap data resources:**
For hub template issues (fromConfigMap), add or correct keys in the
ConfigMap that the template references.

**Strategy E — ConfigMap patch overlay (upstream CR override):**
Only needed when an upstream source-crs template has incorrect defaults
that cannot be fixed via the PolicyGenerator patches field.

---

## Level 5: Implement and Confirm

1. Verify no changed files are inside source-crs/
2. Present the proposed fix (files, diff, strategy) to the user
3. Wait for explicit user confirmation
4. Commit and push using git-mcp-server

After applying: monitor ArgoCD sync and policy compliance status.

---

## Report Template

- **Policy:** (name)
- **Affected Clusters:** (list)
- **Root Cause:** (category and description)
- **Fix Strategy:** A/B/C/D/E with rationale
- **Changes Made:** (file paths and description)
- **Result:** Compliant / Pending sync / Requires manual intervention
`, policyName, clusterNote, metadataYaml, policyYaml, complianceYaml, placementYaml, argoAppYaml)
}

type argoAppRef struct {
	name      string
	namespace string
}

func findPolicies(ctx context.Context, dynamicClient dynamic.Interface, policyName, namespace string) []unstructured.Unstructured {
	var policies []unstructured.Unstructured

	if namespace != "" {
		policy, err := dynamicClient.Resource(policyGVR).Namespace(namespace).Get(ctx, policyName, metav1.GetOptions{})
		if err != nil {
			list, listErr := dynamicClient.Resource(policyGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if listErr != nil {
				return nil
			}
			for _, p := range list.Items {
				if strings.Contains(p.GetName(), policyName) {
					policies = append(policies, p)
				}
			}
		} else {
			policies = []unstructured.Unstructured{*policy}
		}
	} else {
		list, err := dynamicClient.Resource(policyGVR).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil
		}
		for _, p := range list.Items {
			if strings.Contains(p.GetName(), policyName) {
				policies = append(policies, p)
			}
		}
	}
	return policies
}

func renderPolicyStatus(policies []unstructured.Unstructured) string {
	if len(policies) == 0 {
		return "### Policy Status\n\n*No matching policies found*\n"
	}
	var result strings.Builder
	result.WriteString("### Policy Status\n\n")
	for _, p := range policies {
		status, found, err := unstructured.NestedMap(p.Object, "status")
		if err != nil || !found {
			result.WriteString(fmt.Sprintf("#### %s/%s\n\n*No status available*\n\n", p.GetNamespace(), p.GetName()))
			continue
		}
		yamlStr, err := output.MarshalYaml(status)
		if err != nil {
			result.WriteString(fmt.Sprintf("#### %s/%s\n\n*Error marshaling status: %v*\n\n", p.GetNamespace(), p.GetName(), err))
			continue
		}
		result.WriteString(fmt.Sprintf("#### %s/%s\n\n```yaml\n%s```\n\n", p.GetNamespace(), p.GetName(), yamlStr))
	}
	return result.String()
}

func extractPolicyMetadata(policies []unstructured.Unstructured) string {
	if len(policies) == 0 {
		return "### Policy Metadata\n\n*No matching policies found*\n"
	}

	var result strings.Builder
	result.WriteString("### Policy Metadata\n\n")

	seen := make(map[string]bool)
	for _, p := range policies {
		ns := p.GetNamespace()
		name := p.GetName()
		key := ns + "/" + name

		if strings.Contains(name, ".") {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		remediation, _, _ := unstructured.NestedString(p.Object, "spec", "remediationAction")
		if remediation == "" {
			remediation = "not set"
		}

		result.WriteString(fmt.Sprintf("#### %s/%s\n\n", ns, name))
		result.WriteString(fmt.Sprintf("- Remediation: %s\n", remediation))

		templates, found, _ := unstructured.NestedSlice(p.Object, "spec", "policy-templates")
		if !found {
			result.WriteString("\n")
			continue
		}

		for _, tmpl := range templates {
			tmplMap, ok := tmpl.(map[string]interface{})
			if !ok {
				continue
			}
			tmplName, _, _ := unstructured.NestedString(tmplMap, "objectDefinition", "metadata", "name")
			tmplRemediation, _, _ := unstructured.NestedString(tmplMap, "objectDefinition", "spec", "remediationAction")
			evalCompliant, _, _ := unstructured.NestedString(tmplMap, "objectDefinition", "spec", "evaluationInterval", "compliant")
			evalNoncompliant, _, _ := unstructured.NestedString(tmplMap, "objectDefinition", "spec", "evaluationInterval", "noncompliant")

			if tmplName == "" {
				continue
			}

			info := fmt.Sprintf("- Template: %s", tmplName)
			if tmplRemediation != "" {
				info += fmt.Sprintf(" | remediation=%s", tmplRemediation)
			}
			if evalCompliant != "" || evalNoncompliant != "" {
				info += fmt.Sprintf(" | evalInterval(compliant=%s, noncompliant=%s)", evalCompliant, evalNoncompliant)
			}
			result.WriteString(info + "\n")

			objTemplates, objFound, _ := unstructured.NestedSlice(tmplMap, "objectDefinition", "spec", "object-templates")
			if !objFound {
				continue
			}
			for _, objTmpl := range objTemplates {
				objMap, ok := objTmpl.(map[string]interface{})
				if !ok {
					continue
				}
				complianceType, _, _ := unstructured.NestedString(objMap, "complianceType")
				objKind, _, _ := unstructured.NestedString(objMap, "objectDefinition", "kind")
				objName, _, _ := unstructured.NestedString(objMap, "objectDefinition", "metadata", "name")
				objNS, _, _ := unstructured.NestedString(objMap, "objectDefinition", "metadata", "namespace")
				if objKind == "" {
					continue
				}
				detail := fmt.Sprintf("  - %s %s/%s", objKind, objNS, objName)
				if complianceType != "" {
					detail += fmt.Sprintf(" [%s]", complianceType)
				}
				result.WriteString(detail + "\n")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

func argoAppFromPolicies(policies []unstructured.Unstructured) *argoAppRef {
	for _, p := range policies {
		labels := p.GetLabels()
		annotations := p.GetAnnotations()

		if appName, ok := labels["app.kubernetes.io/instance"]; ok && appName != "" {
			return &argoAppRef{name: appName, namespace: ""}
		}
		if appName, ok := annotations["argocd.argoproj.io/instance"]; ok && appName != "" {
			return &argoAppRef{name: appName, namespace: ""}
		}
		if trackingID, ok := annotations["argocd.argoproj.io/tracking-id"]; ok && trackingID != "" {
			parts := strings.SplitN(trackingID, ":", 2)
			if len(parts) >= 1 && parts[0] != "" {
				return &argoAppRef{name: parts[0], namespace: ""}
			}
		}
	}
	return nil
}

func fetchComplianceDetails(ctx context.Context, dynamicClient dynamic.Interface, policies []unstructured.Unstructured, cluster string) string {
	var result strings.Builder
	result.WriteString("### Compliance Details\n\n")

	foundData := false

	for _, p := range policies {
		pNS := p.GetNamespace()
		pName := p.GetName()

		statusSlice, found, _ := unstructured.NestedSlice(p.Object, "status", "status")
		if !found {
			renderReplicatedDetails(&result, &p, cluster)
			foundData = true
			continue
		}

		for _, entry := range statusSlice {
			entryMap, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			clusterName, _, _ := unstructured.NestedString(entryMap, "clustername")
			if cluster != "" && clusterName != cluster {
				continue
			}
			compliance, _, _ := unstructured.NestedString(entryMap, "compliant")

			result.WriteString(fmt.Sprintf("#### Cluster: %s — %s\n\n", clusterName, compliance))
			foundData = true

			replicatedName := fmt.Sprintf("%s.%s", pNS, pName)
			replicatedPolicy, err := dynamicClient.Resource(policyGVR).Namespace(clusterName).Get(ctx, replicatedName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			renderReplicatedDetails(&result, replicatedPolicy, "")
		}
	}

	if !foundData {
		result.WriteString("*No compliance data found — policy may not be placed on any cluster*\n")
	}

	return result.String()
}

func renderReplicatedDetails(w *strings.Builder, p *unstructured.Unstructured, cluster string) {
	clusterName := p.GetNamespace()
	if cluster != "" && clusterName != cluster {
		return
	}

	details, found, _ := unstructured.NestedSlice(p.Object, "status", "details")
	if !found {
		return
	}

	for _, detail := range details {
		detailMap, ok := detail.(map[string]interface{})
		if !ok {
			continue
		}

		templateName, _, _ := unstructured.NestedString(detailMap, "templateMeta", "name")
		templateCompliance, _, _ := unstructured.NestedString(detailMap, "compliant")

		fmt.Fprintf(w, "**Template %s**: %s\n", templateName, templateCompliance)

		history, hFound, _ := unstructured.NestedSlice(detailMap, "history")
		if !hFound || len(history) == 0 {
			w.WriteString("\n")
			continue
		}

		limit := 5
		if len(history) < limit {
			limit = len(history)
		}

		for i := 0; i < limit; i++ {
			histEntry, ok := history[i].(map[string]interface{})
			if !ok {
				continue
			}
			timestamp, _, _ := unstructured.NestedString(histEntry, "lastTimestamp")
			message, _, _ := unstructured.NestedString(histEntry, "message")
			fmt.Fprintf(w, "- [%s] %s\n", timestamp, message)
		}
		w.WriteString("\n")
	}
}

func fetchPlacementInfo(ctx context.Context, dynamicClient dynamic.Interface, policies []unstructured.Unstructured, namespace string) string {
	var result strings.Builder
	result.WriteString("### Placement Info\n\n")

	if namespace == "" {
		namespace = "ztp-policies"
	}

	policyNames := make(map[string]bool)
	for _, p := range policies {
		if !strings.Contains(p.GetName(), ".") {
			policyNames[p.GetName()] = true
		}
	}

	list, err := dynamicClient.Resource(placementBindingGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.WriteString(fmt.Sprintf("*Error listing placement bindings in %s: %v*\n", namespace, err))
		return result.String()
	}

	found := false
	for _, pb := range list.Items {
		subjects, subFound, _ := unstructured.NestedSlice(pb.Object, "subjects")
		if !subFound {
			continue
		}

		relevant := false
		for _, sub := range subjects {
			subMap, ok := sub.(map[string]interface{})
			if !ok {
				continue
			}
			subName, _, _ := unstructured.NestedString(subMap, "name")
			if policyNames[subName] {
				relevant = true
				break
			}
		}
		if !relevant {
			continue
		}

		found = true
		yamlStr, err := output.MarshalYaml(pb.Object)
		if err != nil {
			result.WriteString(fmt.Sprintf("*Error marshaling %s: %v*\n\n", pb.GetName(), err))
			continue
		}
		result.WriteString(fmt.Sprintf("#### %s\n\n```yaml\n%s```\n\n", pb.GetName(), yamlStr))

		placementName, _, _ := unstructured.NestedString(pb.Object, "placementRef", "name")
		if placementName == "" {
			continue
		}

		decisionList, err := dynamicClient.Resource(placementDecisionGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("cluster.open-cluster-management.io/placement=%s", placementName),
		})
		if err != nil || len(decisionList.Items) == 0 {
			result.WriteString(fmt.Sprintf("*Placement %s: no decisions found (0 clusters selected)*\n\n", placementName))
			continue
		}

		var clusterNames []string
		for _, dec := range decisionList.Items {
			decisions, dFound, _ := unstructured.NestedSlice(dec.Object, "status", "decisions")
			if !dFound {
				continue
			}
			for _, d := range decisions {
				dMap, ok := d.(map[string]interface{})
				if !ok {
					continue
				}
				cn, _, _ := unstructured.NestedString(dMap, "clusterName")
				if cn != "" {
					clusterNames = append(clusterNames, cn)
				}
			}
		}
		if len(clusterNames) > 0 {
			result.WriteString(fmt.Sprintf("*Placement %s selects %d cluster(s): %s*\n\n",
				placementName, len(clusterNames), strings.Join(clusterNames, ", ")))
		} else {
			result.WriteString(fmt.Sprintf("*Placement %s selects 0 clusters — policy is NOT applied anywhere*\n\n", placementName))
		}
	}

	if !found {
		result.WriteString(fmt.Sprintf("*No relevant placement bindings found in namespace %s*\n", namespace))
	}

	return result.String()
}

func fetchArgoApps(ctx context.Context, dynamicClient dynamic.Interface, ref *argoAppRef) string {
	var result strings.Builder
	result.WriteString("### ArgoCD Application\n\n")

	if ref != nil && ref.name != "" {
		app := findArgoAppByName(ctx, dynamicClient, ref)
		if app != nil {
			renderArgoApp(&result, app)
			return result.String()
		}
	}

	list, err := dynamicClient.Resource(argoCDAppGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.WriteString(fmt.Sprintf("*Error listing ArgoCD applications: %v*\n", err))
		return result.String()
	}

	keywords := []string{"polic", "ztp", "site", "fleet", "gitops"}
	appFound := false
	for _, app := range list.Items {
		combined := strings.ToLower(app.GetName() + "/" + app.GetNamespace())
		matched := false
		for _, kw := range keywords {
			if strings.Contains(combined, kw) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		renderArgoApp(&result, &app)
		appFound = true
	}

	if !appFound {
		result.WriteString("*No ArgoCD application found for this policy. Ask the user for the git repository URL.*\n")
	}
	return result.String()
}

func findArgoAppByName(ctx context.Context, dynamicClient dynamic.Interface, ref *argoAppRef) *unstructured.Unstructured {
	if ref.namespace != "" {
		app, err := dynamicClient.Resource(argoCDAppGVR).Namespace(ref.namespace).Get(ctx, ref.name, metav1.GetOptions{})
		if err == nil {
			return app
		}
	}
	list, err := dynamicClient.Resource(argoCDAppGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	for i, app := range list.Items {
		if app.GetName() == ref.name {
			return &list.Items[i]
		}
	}
	return nil
}

func renderArgoApp(w *strings.Builder, app *unstructured.Unstructured) {
	name := app.GetName()
	ns := app.GetNamespace()

	repoURL, _, _ := unstructured.NestedString(app.Object, "spec", "source", "repoURL")
	sourcePath, _, _ := unstructured.NestedString(app.Object, "spec", "source", "path")
	targetRevision, _, _ := unstructured.NestedString(app.Object, "spec", "source", "targetRevision")
	syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
	healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
	syncRevision, _, _ := unstructured.NestedString(app.Object, "status", "sync", "revision")

	fmt.Fprintf(w, "#### %s/%s\n\n", ns, name)
	if repoURL != "" {
		fmt.Fprintf(w, "- **repoURL**: %s\n", repoURL)
	}
	if sourcePath != "" {
		fmt.Fprintf(w, "- **path**: %s\n", sourcePath)
	}
	if targetRevision != "" {
		fmt.Fprintf(w, "- **targetRevision**: %s\n", targetRevision)
	}
	if syncStatus != "" {
		fmt.Fprintf(w, "- **sync**: %s\n", syncStatus)
	}
	if healthStatus != "" {
		fmt.Fprintf(w, "- **health**: %s\n", healthStatus)
	}
	if syncRevision != "" {
		fmt.Fprintf(w, "- **deployedRevision**: %s\n", syncRevision)
	}
	w.WriteString("\n")
}
