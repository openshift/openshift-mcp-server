package netedge

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func (s *NetEdgeTestSuite) TestGetCoreDNSConfig() {

	tests := []struct {
		name           string
		configMap      *corev1.ConfigMap
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name: "success - corefile found",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dns-default",
					Namespace: "openshift-dns",
				},
				Data: map[string]string{
					"Corefile": "example corefile content",
				},
			},
			expectedOutput: "example corefile content",
			expectError:    false,
		},
		{
			name:           "failure - configmap not found",
			configMap:      nil,
			expectedOutput: "",
			expectError:    true,
			errorContains:  "failed to get dns-default ConfigMap",
		},
		{
			name: "failure - Corefile key missing",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dns-default",
					Namespace: "openshift-dns",
				},
				Data: map[string]string{
					"OtherData": "some data",
				},
			},
			expectedOutput: "",
			expectError:    true,
			errorContains:  "corefile not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mock client
			objs := []runtime.Object{}
			if tt.configMap != nil {
				objs = append(objs, tt.configMap)
			}

			// Override newClientFunc to return fake client
			originalNewClientFunc := newClientFunc
			defer func() { newClientFunc = originalNewClientFunc }()

			newClientFunc = func(config *rest.Config, options client.Options) (client.Client, error) {
				return fake.NewClientBuilder().WithRuntimeObjects(objs...).Build(), nil
			}

			// Call handler using suite params (which has valid RESTConfig from SetupTest)
			result, err := getCoreDNSConfig(s.params)

			if tt.expectError {
				s.Require().NoError(err) // Handler returns error in result, not as return value
				s.Require().NotNil(result)
				s.Require().Error(result.Error)
				s.Assert().Contains(result.Error.Error(), tt.errorContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				s.Require().NoError(result.Error)
				s.Assert().Equal(tt.expectedOutput, result.Content)
			}
		})
	}
}
