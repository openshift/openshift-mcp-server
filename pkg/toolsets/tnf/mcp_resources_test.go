package tnf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitMCPResources(t *testing.T) {
	resources := initMCPResources()
	require.Len(t, resources, 1)

	r := resources[0]
	assert.Equal(t, "tnf://domain-knowledge/fencing", r.Resource.URI)
	assert.Equal(t, "tnf-fencing-domain-knowledge", r.Resource.Name)
	assert.Equal(t, "text/markdown", r.Resource.MIMEType)
	assert.NotNil(t, r.Handler)
}

func TestFencingDomainKnowledgeResource(t *testing.T) {
	resources := initMCPResources()
	require.Len(t, resources, 1)

	content, err := resources[0].Handler(context.Background())
	require.NoError(t, err)
	require.NotNil(t, content)
	assert.NotEmpty(t, content.Text)

	assert.Contains(t, content.Text, "Split-Brain Risk Assessment")
	assert.Contains(t, content.Text, "STONITH")
	assert.Contains(t, content.Text, "quorum")
	assert.Contains(t, content.Text, "fence_ipmilan")
	assert.Contains(t, content.Text, "Recovery")
	assert.Contains(t, content.Text, "pcs property set stonith-enabled=true")
}
