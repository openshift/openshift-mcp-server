package netedge

import (
	"encoding/json"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func (s *NetEdgeTestSuite) TestGetServiceEndpoints() {

	tests := []struct {
		name          string
		namespace     string
		service       string
		existingObjs  []runtime.Object
		expectedError string
		validate      func(result string)
	}{
		{
			name:      "successful retrieval",
			namespace: "default",
			service:   "myservice",
			existingObjs: []runtime.Object{
				&discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "myservice-1",
						Namespace: "default",
						Labels: map[string]string{
							"kubernetes.io/service-name": "myservice",
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"1.2.3.4"},
						},
					},
					Ports: []discoveryv1.EndpointPort{
						{
							Port: ptr.To(int32(80)),
						},
					},
				},
			},
			validate: func(result string) {
				var eps []discoveryv1.EndpointSlice
				err := json.Unmarshal([]byte(result), &eps)
				s.Require().NoError(err)
				s.Assert().Len(eps, 1)
				s.Assert().Equal("myservice-1", eps[0].Name)
				s.Assert().Equal("1.2.3.4", eps[0].Endpoints[0].Addresses[0])
			},
		},
		{
			name:          "endpoints not found",
			namespace:     "default",
			service:       "missing",
			existingObjs:  []runtime.Object{},
			expectedError: "no EndpointSlices found",
		},
		{
			name:          "missing arguments",
			namespace:     "",
			service:       "",
			expectedError: "parameter required",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create fake dynamic client
			scheme := runtime.NewScheme()
			_ = clientgoscheme.AddToScheme(scheme)
			_ = discoveryv1.AddToScheme(scheme)
			dynClient := fake.NewSimpleDynamicClient(scheme, tt.existingObjs...)

			// Create mock params
			args := make(map[string]any)
			if tt.namespace != "" {
				args["namespace"] = tt.namespace
			}
			if tt.service != "" {
				args["service"] = tt.service
			}

			s.SetArgs(args)
			s.SetDynamicClient(dynClient)

			result, err := getServiceEndpoints(s.params)

			// If the handler returns an error in ToolCallResult (which is mostly what it does for logic errors),
			// result.Error will be set. `err` return is usually nil unless panic/protocol error.

			// However, our handler returns `api.NewToolCallResult("", err)`.
			// So we check result.Error.

			if tt.expectedError != "" {
				s.Assert().NoError(err) // The handler doesn't return Go error
				s.Require().NotNil(result)
				s.Require().Error(result.Error)
				s.Assert().Contains(result.Error.Error(), tt.expectedError)
			} else {
				s.Assert().NoError(err)
				s.Require().NotNil(result)
				s.Assert().NoError(result.Error)
				if tt.validate != nil {
					tt.validate(result.Content)
				}
			}
		})
	}
}
