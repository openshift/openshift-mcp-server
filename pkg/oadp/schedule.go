package oadp

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetSchedule retrieves a Schedule by namespace and name
func GetSchedule(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(ScheduleGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListSchedules lists all schedules in a namespace
func ListSchedules(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(ScheduleGVR).Namespace(namespace).List(ctx, opts)
}

// CreateSchedule creates a new schedule
func CreateSchedule(ctx context.Context, client dynamic.Interface, schedule *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := schedule.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(ScheduleGVR).Namespace(namespace).Create(ctx, schedule, metav1.CreateOptions{})
}

// DeleteSchedule deletes a schedule
func DeleteSchedule(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(ScheduleGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// UpdateSchedule updates an existing schedule
func UpdateSchedule(ctx context.Context, client dynamic.Interface, schedule *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := schedule.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(ScheduleGVR).Namespace(namespace).Update(ctx, schedule, metav1.UpdateOptions{})
}

// PauseSchedule pauses or unpauses a schedule
func PauseSchedule(ctx context.Context, client dynamic.Interface, namespace, name string, paused bool) (*unstructured.Unstructured, error) {
	schedule, err := GetSchedule(ctx, client, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := unstructured.SetNestedField(schedule.Object, paused, "spec", "paused"); err != nil {
		return nil, fmt.Errorf("failed to set paused field: %w", err)
	}

	return client.Resource(ScheduleGVR).Namespace(namespace).Update(ctx, schedule, metav1.UpdateOptions{})
}

// GetScheduleStatus retrieves schedule status information
func GetScheduleStatus(schedule *unstructured.Unstructured) (lastBackup string, paused bool, err error) {
	lastBackup, _, _ = unstructured.NestedString(schedule.Object, "status", "lastBackup")
	paused, _, err = unstructured.NestedBool(schedule.Object, "spec", "paused")
	return lastBackup, paused, err
}
