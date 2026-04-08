package ocp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	schemek8s "k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

// NodeDebugTestEnv bundles a test client with a controllable pods client for tests.
type NodeDebugTestEnv struct {
	Client *FakeNodeDebugClient
	Pods   *FakePodInterface
}

// NewNodeDebugTestEnv constructs a testing harness for exercising NodesDebugExec.
func NewNodeDebugTestEnv(t *testing.T) *NodeDebugTestEnv {
	t.Helper()

	podsClient := &FakePodInterface{}
	fakeClient := &FakeNodeDebugClient{
		pods:      podsClient,
		namespace: "default",
	}

	return &NodeDebugTestEnv{
		Client: fakeClient,
		Pods:   podsClient,
	}
}

// FakeNodeDebugClient implements the NodeDebugClient interface for testing.
type FakeNodeDebugClient struct {
	pods      *FakePodInterface
	namespace string
}

func (f *FakeNodeDebugClient) NamespaceOrDefault(namespace string) string {
	if namespace == "" {
		return f.namespace
	}
	return namespace
}

func (f *FakeNodeDebugClient) Pods(_ string) corev1client.PodInterface {
	return f.pods
}

func (f *FakeNodeDebugClient) PodsLog(ctx context.Context, namespace, name, container string, previous bool, tail int64) (string, error) {
	req := f.pods.GetLogs(name, &corev1.PodLogOptions{Container: container, Previous: previous})
	res := req.Do(ctx)
	if res.Error() != nil {
		return "", res.Error()
	}
	rawData, err := res.Raw()
	if err != nil {
		return "", err
	}
	return string(rawData), nil
}

// FakePodInterface implements corev1client.PodInterface with deterministic behaviour for tests.
type FakePodInterface struct {
	corev1client.PodInterface
	Created           *corev1.Pod
	Deleted           bool
	ExitCode          int32
	TerminatedReason  string
	TerminatedMessage string
	WaitingReason     string
	WaitingMessage    string
	Logs              string
}

func (f *FakePodInterface) Create(_ context.Context, pod *corev1.Pod, _ metav1.CreateOptions) (*corev1.Pod, error) {
	copy := pod.DeepCopy()
	if copy.Name == "" && copy.GenerateName != "" {
		copy.Name = copy.GenerateName + "test"
	}
	f.Created = copy
	return copy.DeepCopy(), nil
}

func (f *FakePodInterface) Get(_ context.Context, _ string, _ metav1.GetOptions) (*corev1.Pod, error) {
	if f.Created == nil {
		return nil, fmt.Errorf("pod not created yet")
	}
	pod := f.Created.DeepCopy()

	// If waiting state is set, return that instead of terminated
	if f.WaitingReason != "" {
		waiting := &corev1.ContainerStateWaiting{Reason: f.WaitingReason}
		if f.WaitingMessage != "" {
			waiting.Message = f.WaitingMessage
		}
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			Name:  NodeDebugContainerName,
			State: corev1.ContainerState{Waiting: waiting},
		}}
		pod.Status.Phase = corev1.PodPending
		return pod, nil
	}

	// Otherwise return terminated state
	terminated := &corev1.ContainerStateTerminated{ExitCode: f.ExitCode}
	if f.TerminatedReason != "" {
		terminated.Reason = f.TerminatedReason
	}
	if f.TerminatedMessage != "" {
		terminated.Message = f.TerminatedMessage
	}
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  NodeDebugContainerName,
		State: corev1.ContainerState{Terminated: terminated},
	}}
	pod.Status.Phase = corev1.PodSucceeded
	return pod, nil
}

func (f *FakePodInterface) Delete(_ context.Context, _ string, _ metav1.DeleteOptions) error {
	f.Deleted = true
	return nil
}

func (f *FakePodInterface) GetLogs(name string, opts *corev1.PodLogOptions) *restclient.Request {
	body := io.NopCloser(strings.NewReader(f.Logs))
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})}
	content := restclient.ClientContentConfig{
		ContentType:  runtime.ContentTypeJSON,
		GroupVersion: schema.GroupVersion{Version: "v1"},
		Negotiator:   runtime.NewClientNegotiator(schemek8s.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"}),
	}
	return restclient.NewRequestWithClient(&url.URL{Scheme: "https", Host: "localhost"}, "", content, client).Verb("GET")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
