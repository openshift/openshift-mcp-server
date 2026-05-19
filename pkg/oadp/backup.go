package oadp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetBackup retrieves a Backup by namespace and name
func GetBackup(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(BackupGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListBackups lists all backups in a namespace
func ListBackups(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(BackupGVR).Namespace(namespace).List(ctx, opts)
}

// CreateBackup creates a new backup
func CreateBackup(ctx context.Context, client dynamic.Interface, backup *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := backup.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(BackupGVR).Namespace(namespace).Create(ctx, backup, metav1.CreateOptions{})
}

// DeleteBackup deletes a backup by creating a DeleteBackupRequest
// This is the proper way to delete a backup as it also removes data from object storage
func DeleteBackup(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	// Create a DeleteBackupRequest which will trigger the Velero controller
	// to delete the backup and its data from object storage
	deleteRequest := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "DeleteBackupRequest",
			"metadata": map[string]any{
				"name":      name + "-delete-" + strconv.FormatInt(time.Now().UnixMilli(), 10),
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupName": name,
			},
		},
	}

	_, err := client.Resource(DeleteBackupRequestGVR).Namespace(namespace).Create(ctx, deleteRequest, metav1.CreateOptions{})
	return err
}

// GetBackupPhase retrieves the backup phase/status
func GetBackupPhase(backup *unstructured.Unstructured) (string, bool, error) {
	return unstructured.NestedString(backup.Object, "status", "phase")
}

// GetBackupStatus retrieves detailed backup status information including phase, errors, warnings, and timestamps
func GetBackupStatus(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	backup, err := GetBackup(ctx, client, namespace, name)
	if err != nil {
		return "", fmt.Errorf("failed to get backup: %w", err)
	}

	phase, _, err := GetBackupPhase(backup)
	if err != nil {
		return "", fmt.Errorf("failed to read backup phase: %w", err)
	}
	errors, _, err := unstructured.NestedInt64(backup.Object, "status", "errors")
	if err != nil {
		return "", fmt.Errorf("failed to read backup errors: %w", err)
	}
	warnings, _, err := unstructured.NestedInt64(backup.Object, "status", "warnings")
	if err != nil {
		return "", fmt.Errorf("failed to read backup warnings: %w", err)
	}
	startTime, _, err := unstructured.NestedString(backup.Object, "status", "startTimestamp")
	if err != nil {
		return "", fmt.Errorf("failed to read backup start time: %w", err)
	}
	completionTime, _, err := unstructured.NestedString(backup.Object, "status", "completionTimestamp")
	if err != nil {
		return "", fmt.Errorf("failed to read backup completion time: %w", err)
	}
	failureReason, _, err := unstructured.NestedString(backup.Object, "status", "failureReason")
	if err != nil {
		return "", fmt.Errorf("failed to read backup failure reason: %w", err)
	}

	result := fmt.Sprintf("Backup: %s/%s\nPhase: %s\nStart Time: %s\nCompletion Time: %s\nErrors: %d\nWarnings: %d",
		namespace, name, phase, startTime, completionTime, errors, warnings)

	if failureReason != "" {
		result += fmt.Sprintf("\nFailure Reason: %s", failureReason)
	}

	return result, nil
}
