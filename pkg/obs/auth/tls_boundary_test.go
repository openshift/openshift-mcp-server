package auth_test

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/obs/auth"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/rest"
)

type TLSBoundarySuite struct{ suite.Suite }

func (s *TLSBoundarySuite) TestCertificateVerificationAndRejectedOptOut() {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := seed.TLS.Certificates[0]
	seed.Close()
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	s.Require().NoError(err)
	for _, test := range []struct {
		name     string
		trusted  bool
		insecure bool
		expired  bool
		wrongDNS bool
		accepted bool
	}{
		{name: "default rejects unknown authority"},
		{name: "configured CA accepts valid certificate", trusted: true, accepted: true},
		{name: "configured CA rejects expired certificate", trusted: true, expired: true},
		{name: "configured CA rejects wrong hostname", trusted: true, wrongDNS: true},
		{name: "explicit insecure rejects unknown authority", insecure: true},
		{name: "explicit insecure rejects expired certificate", trusted: true, insecure: true, expired: true},
		{name: "explicit insecure rejects wrong hostname", trusted: true, insecure: true, wrongDNS: true},
		{name: "explicit insecure rejects even trusted certificate", trusted: true, insecure: true},
	} {
		s.Run(test.name, func() {
			template := *leaf
			template.SerialNumber = big.NewInt(2)
			template.NotBefore = time.Now().Add(-2 * time.Hour)
			template.NotAfter = time.Now().Add(time.Hour)
			if test.expired {
				template.NotAfter = time.Now().Add(-time.Hour)
			}
			if test.wrongDNS {
				template.IPAddresses = nil
				template.DNSNames = []string{"wrong.invalid"}
			}
			der, err := x509.CreateCertificate(rand.Reader, &template, &template, leaf.PublicKey, certificate.PrivateKey)
			s.Require().NoError(err)
			var requests atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				response.Header().Set("X-Received-Auth", request.Header.Get("Authorization"))
				response.WriteHeader(http.StatusOK)
			}))
			server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: certificate.PrivateKey}}}
			server.StartTLS()
			defer server.Close()
			config := &rest.Config{BearerToken: "owned-fixture-token"}
			if test.trusted {
				config.CAData = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
			}
			transport, err := auth.BuildRoundTripper(s.T().Context(), config, auth.AuthModeKubeConfig, true, test.insecure)
			if test.insecure {
				s.Require().ErrorContains(err, "insecure=true is no longer supported")
				s.Nil(transport)
				s.Equal(int32(0), requests.Load())
				return
			}
			s.Require().NoError(err)
			request, err := http.NewRequestWithContext(s.T().Context(), http.MethodGet, server.URL+"/api/v1/query", http.NoBody)
			s.Require().NoError(err)
			response, err := transport.RoundTrip(request)
			if !test.accepted {
				s.Require().Error(err)
				s.Nil(response)
				s.Equal(int32(0), requests.Load(), "a rejected TLS peer must never receive the bearer token")
				return
			}
			s.Require().NoError(err)
			s.Require().NoError(response.Body.Close())
			s.Equal("Bearer owned-fixture-token", response.Header.Get("X-Received-Auth"))
			s.Equal(int32(1), requests.Load())
		})
	}
}

func TestTLSBoundary(t *testing.T) {
	suite.Run(t, new(TLSBoundarySuite))
}
