package kubernetes

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
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

// NodeDebugTestEnv bundles a Kubernetes instance with a controllable pods client for tests.
type NodeDebugTestEnv struct {
	Kubernetes *Kubernetes
	Pods       *FakePodInterface
}

// NewNodeDebugTestEnv constructs a testing harness for exercising NodesDebugExec.
func NewNodeDebugTestEnv(t *testing.T) *NodeDebugTestEnv {
	t.Helper()

	podsClient := &FakePodInterface{}
	clientConfig := clientcmd.NewDefaultClientConfig(*clientcmdapi.NewConfig(), &clientcmd.ConfigOverrides{})
	manager := &Manager{
		staticConfig:    &config.StaticConfig{},
		clientCmdConfig: clientConfig,
	}
	k := &Kubernetes{manager: manager}
	k.podClientFactory = func(namespace string) (corev1client.PodInterface, error) {
		podsClient.namespace = namespace
		return podsClient, nil
	}

	return &NodeDebugTestEnv{
		Kubernetes: k,
		Pods:       podsClient,
	}
}

// FakePodInterface implements corev1client.PodInterface with deterministic behaviour for tests.
type FakePodInterface struct {
	corev1client.PodInterface
	namespace         string
	Created           *corev1.Pod
	Deleted           bool
	ExitCode          int32
	TerminatedReason  string
	TerminatedMessage string
	Logs              string
}

func (f *FakePodInterface) Create(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions) (*corev1.Pod, error) {
	copy := pod.DeepCopy()
	if copy.Name == "" && copy.GenerateName != "" {
		copy.Name = copy.GenerateName + "test"
	}
	f.Created = copy
	return copy.DeepCopy(), nil
}

func (f *FakePodInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error) {
	if f.Created == nil {
		return nil, fmt.Errorf("pod not created yet")
	}
	pod := f.Created.DeepCopy()
	terminated := &corev1.ContainerStateTerminated{ExitCode: f.ExitCode}
	if f.TerminatedReason != "" {
		terminated.Reason = f.TerminatedReason
	}
	if f.TerminatedMessage != "" {
		terminated.Message = f.TerminatedMessage
	}
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  nodeDebugContainerName,
		State: corev1.ContainerState{Terminated: terminated},
	}}
	pod.Status.Phase = corev1.PodSucceeded
	return pod, nil
}

func (f *FakePodInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
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
