package oadp

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
)

// backupDefaults holds auto-discovered configuration for backup creation.
type backupDefaults struct {
	// StorageLocation is the name of the default/available BSL to use
	StorageLocation string
	// DefaultVolumesToFsBackup indicates NodeAgent is enabled for file-system backup
	DefaultVolumesToFsBackup *bool
	// Warnings contains advisory messages about the OADP environment
	Warnings []string
}

// discoverBackupDefaults inspects the OADP environment and returns
// smart defaults for backup creation. This encodes domain knowledge
// that generic tools don't have:
//   - Auto-discovers the default or first available BSL
//   - Detects backup method from DPA configuration (CSI vs file-system)
//   - Validates that target namespaces exist
func discoverBackupDefaults(ctx context.Context, client dynamic.Interface, oadpNamespace string, includedNamespaces []string) backupDefaults {
	defaults := backupDefaults{}

	// 1. Discover DPA and detect backup method
	dpas, err := oadp.ListDataProtectionApplications(ctx, client, oadpNamespace, metav1.ListOptions{})
	if err != nil {
		defaults.Warnings = append(defaults.Warnings,
			fmt.Sprintf("Failed to check DataProtectionApplication: %v", err))
	} else if len(dpas.Items) == 0 {
		defaults.Warnings = append(defaults.Warnings,
			"No DataProtectionApplication found in namespace "+oadpNamespace+". OADP may not be configured — the backup will likely fail.")
	} else {
		// Check if NodeAgent is enabled (file-system backup via Kopia/Restic)
		dpa := dpas.Items[0]
		nodeAgentEnabled, found, _ := unstructured.NestedBool(dpa.Object, "spec", "configuration", "nodeAgent", "enable")
		if found && nodeAgentEnabled {
			fsBackup := true
			defaults.DefaultVolumesToFsBackup = &fsBackup
		}
	}

	// 2. Auto-discover the default or first available BSL
	bsls, err := oadp.ListBackupStorageLocations(ctx, client, oadpNamespace, metav1.ListOptions{})
	if err != nil {
		defaults.Warnings = append(defaults.Warnings,
			fmt.Sprintf("Failed to check BackupStorageLocations: %v", err))
	} else if len(bsls.Items) == 0 {
		defaults.Warnings = append(defaults.Warnings,
			"No BackupStorageLocation found. Backups require a storage location to be configured.")
	} else {
		defaults.StorageLocation = findDefaultBSL(bsls.Items)
		if defaults.StorageLocation == "" {
			defaults.Warnings = append(defaults.Warnings,
				"No BackupStorageLocation is in 'Available' phase. The backup may fail. Check storage credentials and bucket configuration.")
		}
	}

	// 3. Validate that target namespaces exist
	if len(includedNamespaces) > 0 {
		missing := validateNamespacesExist(ctx, client, includedNamespaces)
		if len(missing) > 0 {
			defaults.Warnings = append(defaults.Warnings,
				fmt.Sprintf("Namespaces not found on cluster: %s. The backup may be empty for these namespaces.", strings.Join(missing, ", ")))
		}
	}

	return defaults
}

// findDefaultBSL finds the best BSL to use:
// 1. A BSL marked as default (spec.default=true) that is Available
// 2. The first BSL in Available phase
// Returns empty string if no suitable BSL found.
func findDefaultBSL(bsls []unstructured.Unstructured) string {
	var firstAvailable string
	for _, bsl := range bsls {
		phase, _, _ := unstructured.NestedString(bsl.Object, "status", "phase")
		if phase != "Available" {
			continue
		}
		// Check if this BSL is marked as default
		isDefault, found, _ := unstructured.NestedBool(bsl.Object, "spec", "default")
		if found && isDefault {
			return bsl.GetName()
		}
		if firstAvailable == "" {
			firstAvailable = bsl.GetName()
		}
	}
	return firstAvailable
}

// validateNamespacesExist checks which of the given namespaces exist on the cluster.
// Returns a list of namespaces that were NOT found. API errors other than NotFound are ignored.
func validateNamespacesExist(ctx context.Context, client dynamic.Interface, namespaces []string) []string {
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	var missing []string
	for _, ns := range namespaces {
		_, err := client.Resource(nsGVR).Get(ctx, ns, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			missing = append(missing, ns)
		}
		// Ignore other errors (RBAC, transient) — don't report them as missing
	}
	return missing
}

// restoreContext holds auto-discovered information about a backup for restore creation.
type restoreContext struct {
	// BackupPhase is the current phase of the backup
	BackupPhase string
	// IncludedNamespaces from the backup spec
	IncludedNamespaces []string
	// IncludedResources from the backup spec
	IncludedResources []string
	// Warnings contains advisory messages
	Warnings []string
}

// discoverRestoreContext inspects the backup to provide context for restore creation.
// This encodes domain knowledge: check backup phase, extract what was backed up.
func discoverRestoreContext(ctx context.Context, client dynamic.Interface, namespace, backupName string) restoreContext {
	rc := restoreContext{}

	backup, err := oadp.GetBackup(ctx, client, namespace, backupName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			rc.Warnings = append(rc.Warnings,
				fmt.Sprintf("Backup '%s' not found in namespace '%s'. The restore may fail. Use oadp_backup with action 'list' to see available backups.", backupName, namespace))
		} else {
			rc.Warnings = append(rc.Warnings,
				fmt.Sprintf("Failed to check backup '%s': %v", backupName, err))
		}
		return rc
	}

	// Extract backup phase
	rc.BackupPhase, _, _ = unstructured.NestedString(backup.Object, "status", "phase")
	switch rc.BackupPhase {
	case "Completed":
		// Good — no warning needed
	case "PartiallyFailed":
		rc.Warnings = append(rc.Warnings,
			fmt.Sprintf("Backup '%s' has phase 'PartiallyFailed'. Some resources may not restore correctly.", backupName))
	case "InProgress":
		rc.Warnings = append(rc.Warnings,
			fmt.Sprintf("Backup '%s' is still in progress. The restore may fail if the backup hasn't completed.", backupName))
	case "Failed":
		rc.Warnings = append(rc.Warnings,
			fmt.Sprintf("Backup '%s' has failed. A restore from a failed backup is not possible.", backupName))
	case "":
		rc.Warnings = append(rc.Warnings,
			fmt.Sprintf("Backup '%s' has no status phase yet. It may still be initializing.", backupName))
	default:
		rc.Warnings = append(rc.Warnings,
			fmt.Sprintf("Backup '%s' has unexpected phase '%s'.", backupName, rc.BackupPhase))
	}

	// Extract what was backed up for context
	if ns, found, _ := unstructured.NestedStringSlice(backup.Object, "spec", "includedNamespaces"); found {
		rc.IncludedNamespaces = ns
	}
	if res, found, _ := unstructured.NestedStringSlice(backup.Object, "spec", "includedResources"); found {
		rc.IncludedResources = res
	}

	return rc
}

// formatWarnings formats a list of warnings into a single string.
func formatWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return "Pre-flight warnings:\n- " + strings.Join(warnings, "\n- ")
}
