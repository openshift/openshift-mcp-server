package mustgather

import (
	"strconv"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
)

type resourceRequest string

func (r resourceRequest) GetURI() string {
	return string(r)
}

func TestResourceCurrentArchiveUsesToolsetConfig(t *testing.T) {
	archiveDir := t.TempDir()
	makeArchiveRoot(t, archiveDir)

	archiveID, err := mg.ArchiveIDFromLocalPath(archiveDir)
	if err != nil {
		t.Fatalf("derive archive ID: %v", err)
	}
	cfg := test.Must(config.ReadToml(t.Context(), []byte(
		"[toolset_configs.\"openshift/mustgather\"]\n"+
			"mustgather_dirs = ["+strconv.Quote(archiveDir)+"]\n",
	)))

	content, err := resourceCurrentArchive(api.ResourceHandlerParams{
		Context: t.Context(),
		Config:  cfg,
		Request: resourceRequest(archiveURIPrefix + archiveID),
	})
	if err != nil {
		t.Fatalf("read archive resource: %v", err)
	}
	if content == nil || content.Text == "" {
		t.Fatal("archive resource returned no text")
	}
	if want := "Path: " + archiveDir + "\n"; !strings.Contains(content.Text, want) {
		t.Fatalf("archive resource does not contain %q:\n%s", want, content.Text)
	}
}
