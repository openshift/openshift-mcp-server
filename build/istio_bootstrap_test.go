package build_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type IstioBootstrapSuite struct{ suite.Suite }

func TestIstioBootstrap(testingT *testing.T) { suite.Run(testingT, new(IstioBootstrapSuite)) }

type istioFixture struct {
	root, checkout, destination, addons, temporary, log string
	environment                                         []string
}

func newIstioFixture(testingT *testing.T, scenario string, literalPath bool) istioFixture {
	testingT.Helper()
	root := testingT.TempDir()
	checkout := filepath.Join(root, "trusted")
	if literalPath {
		checkout += " ' $literal"
	}
	fixture := istioFixture{root: root, checkout: checkout, destination: filepath.Join(checkout, "_output/tools/bin/istioctl"), addons: filepath.Join(checkout, "_output/istio-addons"), temporary: filepath.Join(root, "temp"), log: filepath.Join(root, "events")}
	for _, name := range []string{"build/kiali.mk", "build/install-istio.sh"} {
		data, err := os.ReadFile(workflowBuildPath(testingT, filepath.Base(name)))
		if err != nil {
			testingT.Fatal(err)
		}
		writeActionFixture(testingT, filepath.Join(checkout, name), string(data), 0o600)
	}
	writeActionFixture(testingT, filepath.Join(checkout, "Makefile"), "include build/kiali.mk\n.PHONY: kubectl\nkubectl:\n\t@true\n", 0o600)
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	files := []struct{ name, content string }{
		{"bin/istioctl", "#!/usr/bin/env bash\nset -eu\nprintf 'istioctl %s\\n' \"$*\" >> \"$EVENT_LOG\"\n"},
		{"samples/addons/jaeger.yaml", "kind: Service\nmetadata:\n  name: tracing\n"},
		{"samples/addons/prometheus.yaml", "kind: Deployment\nmetadata:\n  name: prometheus\n"},
		{"samples/addons/kiali.yaml", "external_services:\n  tracing:\n    enabled: false\n  identity:\n    enabled: false\n"},
		{"download-installer.sh", "touch \"$FIXTURE_ROOT/SENTINEL\"\n"},
	}
	for _, file := range files {
		if scenario == "missing-member" && file.name == "samples/addons/kiali.yaml" {
			continue
		}
		header := &tar.Header{Name: "istio-1.31.0/" + file.name, Mode: 0o644, Size: int64(len(file.content)), Typeflag: tar.TypeReg}
		if scenario == "symlink-member" && file.name == "bin/istioctl" {
			header.Typeflag, header.Linkname, header.Size = tar.TypeSymlink, "../../escape", 0
		}
		if err := writer.WriteHeader(header); err != nil {
			testingT.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg {
			if _, err := writer.Write([]byte(file.content)); err != nil {
				testingT.Fatal(err)
			}
		}
		if scenario == "duplicate-member" && file.name == "bin/istioctl" {
			if err := writer.WriteHeader(header); err != nil {
				testingT.Fatal(err)
			}
			if _, err := writer.Write([]byte(file.content)); err != nil {
				testingT.Fatal(err)
			}
		}
	}
	if scenario == "traversal" || scenario == "absolute-member" {
		name := "istio-1.31.0/../escape"
		if scenario == "absolute-member" {
			name = "/escape"
		}
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Typeflag: tar.TypeReg}); err != nil {
			testingT.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		testingT.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		testingT.Fatal(err)
	}
	payload := archive.Bytes()
	if scenario == "invalid-archive" {
		payload = []byte("not an archive")
	}
	writeActionFixture(testingT, filepath.Join(root, "archive.tar.gz"), string(payload), 0o600)
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	if scenario == "checksum" {
		digest = strings.Repeat("0", 64)
	}
	var manifest strings.Builder
	for _, platform := range []string{"linux-amd64", "linux-arm64", "linux-armv7", "osx-amd64", "osx-arm64"} {
		fmt.Fprintf(&manifest, "%s  istio-1.31.0-%s.tar.gz\n", digest, platform)
	}
	manifestData := manifest.String()
	if scenario == "missing-hash" {
		manifestData = strings.Replace(manifestData, digest+"  istio-1.31.0-linux-amd64.tar.gz\n", "", 1)
	}
	if scenario == "duplicate-hash" {
		manifestData += digest + "  istio-1.31.0-linux-amd64.tar.gz\n"
	}
	if scenario == "malformed-hash" {
		manifestData = strings.Replace(manifestData, digest, "invalid", 1)
	}
	if scenario != "missing-manifest" {
		writeActionFixture(testingT, filepath.Join(checkout, "build/istio-1.31.0.sha256"), manifestData, 0o600)
	}
	bin := filepath.Join(root, "mock-bin")
	writeActionFixture(testingT, filepath.Join(bin, "uname"), "#!/bin/bash\ncase $1 in -s) printf '%s' \"$FIXTURE_OS\";; -m) printf '%s' \"$FIXTURE_ARCH\";; *) exit 9;; esac\n", 0o755)
	writeActionFixture(testingT, filepath.Join(bin, "curl"), `#!/bin/bash
set -eu
printf 'download %s\n' "$*" >> "$EVENT_LOG"
[[ ${FAIL_DOWNLOAD:-false} != true ]] || exit 22
[[ $1 == --fail && $5 == --proto && $6 == =https && $7 == --tlsv1.2 && $8 == --connect-timeout && $9 == 10 && ${10} == --max-time && ${11} == 300 && ${13} == --output && $# == 14 ]]
cp "$FIXTURE_ROOT/archive.tar.gz" "${14}"
`, 0o755)
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		testingT.Fatal(err)
	}
	shaPath, err := exec.LookPath("sha256sum")
	if err != nil {
		testingT.Fatal(err)
	}
	writeActionFixture(testingT, filepath.Join(bin, "tar"), "#!/bin/bash\nset -eu\nprintf 'tar %s\\n' \"$*\" >> \"$EVENT_LOG\"\n[[ ${FAIL_EXTRACT:-false} != true || $1 != -xOzf ]] || exit 7\nexec '"+tarPath+"' \"$@\"\n", 0o755)
	writeActionFixture(testingT, filepath.Join(bin, "sha256sum"), "#!/bin/bash\nset -eu\nprintf 'checksum\\n' >> \"$EVENT_LOG\"\nexec '"+shaPath+"' \"$@\"\n", 0o755)
	writeActionFixture(testingT, filepath.Join(bin, "kubectl"), "#!/bin/bash\nset -eu\n[[ $KUBECONFIG == /dev/null && $(command -v kubectl) == \"$FIXTURE_ROOT/mock-bin/kubectl\" ]]\nprintf 'kubectl %s\\n' \"$*\" >> \"$EVENT_LOG\"\n", 0o755)
	if err := os.MkdirAll(fixture.temporary, 0o700); err != nil {
		testingT.Fatal(err)
	}
	fixture.environment = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FIXTURE_ROOT="+root, "EVENT_LOG="+fixture.log, "TMPDIR="+fixture.temporary, "KUBECONFIG=/dev/null", "FIXTURE_OS=Linux", "FIXTURE_ARCH=x86_64")
	return fixture
}

func (fixture istioFixture) run(target string, overrides ...string) (string, error) {
	command := exec.Command("make", target, "KUBECTL="+filepath.Join(fixture.root, "mock-bin/kubectl"))
	command.Dir, command.Env = fixture.checkout, append(fixture.environment, overrides...)
	output, err := command.CombinedOutput()
	return string(output), err
}

func (suiteState *IstioBootstrapSuite) assertClean(fixture istioFixture) {
	entries, err := os.ReadDir(fixture.temporary)
	suiteState.Require().NoError(err)
	suiteState.Empty(entries)
	suiteState.NoFileExists(filepath.Join(fixture.root, "SENTINEL"))
	suiteState.NoFileExists(filepath.Join(fixture.checkout, "escape"))
	suiteState.NoFileExists(filepath.Join(fixture.addons, "download-installer.sh"))
}

func (suiteState *IstioBootstrapSuite) TestActualMakePlatformConsumerAndLiteralPaths() {
	for _, testCase := range []struct{ os, arch, platform string }{
		{"Linux", "x86_64", "linux-amd64"}, {"Linux", "aarch64", "linux-arm64"}, {"Linux", "arm64", "linux-arm64"}, {"Linux", "armv7l", "linux-armv7"}, {"Darwin", "x86_64", "osx-amd64"}, {"Darwin", "arm64", "osx-arm64"},
	} {
		suiteState.Run(testCase.os+"-"+testCase.arch, func() {
			fixture := newIstioFixture(suiteState.T(), "valid", true)
			output, err := fixture.run("istioctl", "FIXTURE_OS="+testCase.os, "FIXTURE_ARCH="+testCase.arch)
			suiteState.Require().NoError(err, output)
			events, err := os.ReadFile(fixture.log)
			suiteState.Require().NoError(err)
			suiteState.Contains(string(events), "https://github.com/istio/istio/releases/download/1.31.0/istio-1.31.0-"+testCase.platform+".tar.gz")
			suiteState.Less(strings.Index(string(events), "checksum\n"), strings.Index(string(events), "tar -tzf"))
			suiteState.NotContains(string(events), "istioctl install")
			for _, name := range []string{"jaeger", "prometheus", "kiali"} {
				info, err := os.Stat(filepath.Join(fixture.addons, name+".yaml"))
				suiteState.Require().NoError(err)
				suiteState.Equal(os.FileMode(0o644), info.Mode().Perm())
			}
			info, err := os.Stat(fixture.destination)
			suiteState.Require().NoError(err)
			suiteState.Equal(os.FileMode(0o755), info.Mode().Perm())
			data, err := os.ReadFile(filepath.Join(fixture.addons, "kiali.yaml"))
			suiteState.Require().NoError(err)
			suiteState.Equal("external_services:\n  tracing:\n    enabled: true\n        in_cluster_url: \"http://tracing.istio-system:16685/jaeger\"\n        use_grpc: true\n  identity:\n    enabled: false\n", string(data))
			suiteState.assertClean(fixture)
		})
	}
}

func (suiteState *IstioBootstrapSuite) TestRejectedInputsNeverPublishOrExecute() {
	for _, scenario := range []string{"checksum", "missing-hash", "duplicate-hash", "malformed-hash", "missing-manifest", "missing-member", "duplicate-member", "symlink-member", "traversal", "absolute-member", "invalid-archive", "download-failure", "extract-failure", "unsupported-os", "unsupported-arch", "destination-directory", "destination-symlink", "parent-symlink", "addons-symlink", "addon-file-symlink"} {
		suiteState.Run(scenario, func() {
			fixture := newIstioFixture(suiteState.T(), scenario, false)
			var overrides []string
			switch scenario {
			case "download-failure":
				overrides = append(overrides, "FAIL_DOWNLOAD=true")
			case "extract-failure":
				overrides = append(overrides, "FAIL_EXTRACT=true")
			case "unsupported-os":
				overrides = append(overrides, "FIXTURE_OS=Windows_NT")
			case "unsupported-arch":
				overrides = append(overrides, "FIXTURE_ARCH=s390x")
			case "destination-directory":
				suiteState.Require().NoError(os.MkdirAll(fixture.destination, 0o700))
			case "destination-symlink":
				suiteState.Require().NoError(os.MkdirAll(filepath.Dir(fixture.destination), 0o700))
				suiteState.Require().NoError(os.Symlink(filepath.Join(fixture.root, "missing"), fixture.destination))
			case "parent-symlink":
				suiteState.Require().NoError(os.Symlink(fixture.root, filepath.Join(fixture.checkout, "_output")))
			case "addons-symlink":
				suiteState.Require().NoError(os.MkdirAll(filepath.Dir(fixture.addons), 0o700))
				suiteState.Require().NoError(os.Symlink(fixture.root, fixture.addons))
			case "addon-file-symlink":
				suiteState.Require().NoError(os.MkdirAll(fixture.addons, 0o700))
				suiteState.Require().NoError(os.Symlink(filepath.Join(fixture.root, "missing"), filepath.Join(fixture.addons, "kiali.yaml")))
			}
			output, err := fixture.run("install-istio", overrides...)
			suiteState.Error(err, output)
			info, statErr := os.Lstat(fixture.destination)
			if statErr == nil {
				suiteState.False(info.Mode().IsRegular(), "failed bootstrap must not publish a binary")
			}
			events, readErr := os.ReadFile(fixture.log)
			if readErr == nil {
				suiteState.NotContains(string(events), "istioctl install")
				suiteState.NotContains(string(events), "kubectl ")
			}
			if strings.Contains(scenario, "hash") || strings.Contains(scenario, "unsupported") || scenario == "missing-manifest" {
				suiteState.NotContains(string(events), "download ")
			}
			if scenario == "checksum" {
				suiteState.NotContains(string(events), "tar ")
			}
			suiteState.assertClean(fixture)
		})
	}
}

func (suiteState *IstioBootstrapSuite) TestExistingSymlinkAncestor() {
	fixture := newIstioFixture(suiteState.T(), "valid", false)
	alias := filepath.Join(fixture.root, "existing-alias")
	suiteState.Require().NoError(os.Symlink(fixture.checkout, alias))
	fixture.checkout = alias
	output, err := fixture.run("istioctl", "PWD="+alias)
	suiteState.Require().NoError(err, output)
	suiteState.FileExists(fixture.destination)
	suiteState.assertClean(fixture)
}

func (suiteState *IstioBootstrapSuite) TestInternalInstallerRejectsEscapes() {
	for _, destination := range []string{"outside", "_output/../escaped", "_output/tools/bin/../istioctl"} {
		suiteState.Run(destination, func() {
			fixture := newIstioFixture(suiteState.T(), "valid", false)
			command := exec.Command("bash", filepath.Join(fixture.checkout, "build/install-istio.sh"), "1.31.0", fixture.checkout+"/"+destination, fixture.addons, filepath.Join(fixture.checkout, "build/istio-1.31.0.sha256"))
			command.Env = fixture.environment
			output, err := command.CombinedOutput()
			suiteState.Error(err, string(output))
			suiteState.NoFileExists(fixture.destination)
			suiteState.NoFileExists(fixture.log)
		})
	}
}

func (suiteState *IstioBootstrapSuite) TestActualMakeProfileWaitsAndPinnedBookinfo() {
	fixture := newIstioFixture(suiteState.T(), "valid", false)
	output, err := fixture.run("install-istio")
	suiteState.Require().NoError(err, output)
	output, err = fixture.run("install-bookinfo-demo")
	suiteState.Require().NoError(err, output)
	events, err := os.ReadFile(fixture.log)
	suiteState.Require().NoError(err)
	for _, expected := range []string{"istioctl install --set profile=demo --set meshConfig.defaultConfig.tracing.zipkin.address=zipkin.istio-system:9411 -y", "kubectl wait --namespace istio-system --for=condition=available deployment/kiali --timeout=300s", "kubectl wait --namespace istio-system --for=condition=available deployment/prometheus --timeout=300s", "kubectl rollout status deployment/kiali -n istio-system", "kubectl label namespace default istio-injection=enabled --overwrite", "kubectl create ns bookinfo", "kubectl label namespace bookinfo istio-discovery=enabled istio.io/rev=default istio-injection=enabled", "https://raw.githubusercontent.com/openshift-service-mesh/istio/b6fb5348dd66280245e3d98d8d93f416ece453c3/samples/bookinfo/platform/kube/bookinfo.yaml", "https://raw.githubusercontent.com/openshift-service-mesh/istio/b6fb5348dd66280245e3d98d8d93f416ece453c3/samples/bookinfo/networking/bookinfo-gateway.yaml", "https://raw.githubusercontent.com/istio-ecosystem/sail-operator/0c7171a9fdc4762366a78e9a0e3f226ff6e6bbd3/chart/samples/ingress-gateway.yaml", "kubectl wait --for=condition=Ready pod --all -n bookinfo --timeout=300s"} {
		suiteState.Contains(string(events), expected)
	}
	suiteState.NotContains(string(events), "refs/heads")
	for _, name := range []string{"prometheus", "kiali", "jaeger"} {
		suiteState.Contains(string(events), "kubectl apply -f "+filepath.Join(fixture.addons, name+".yaml")+" -n istio-system")
	}
	suiteState.assertClean(fixture)
}

func (suiteState *IstioBootstrapSuite) TestExistingBinaryIsNeverTrustedWithoutDownload() {
	for _, scenario := range []string{"valid", "checksum"} {
		suiteState.Run(scenario, func() {
			fixture := newIstioFixture(suiteState.T(), scenario, false)
			writeActionFixture(suiteState.T(), fixture.destination, "#!/bin/bash\ntouch \"$FIXTURE_ROOT/SENTINEL\"\nexit 97\n", 0o755)
			output, err := fixture.run("install-istio")
			if scenario == "valid" {
				suiteState.NoError(err, output)
			} else {
				suiteState.Error(err, output)
			}
			events, err := os.ReadFile(fixture.log)
			suiteState.Require().NoError(err)
			suiteState.Contains(string(events), "download ")
			suiteState.Contains(string(events), "checksum\n")
			suiteState.assertClean(fixture)
		})
	}
}

func (suiteState *IstioBootstrapSuite) TestCommittedManifestAndDocumentation() {
	data, err := os.ReadFile(workflowBuildPath(suiteState.T(), "istio-1.31.0.sha256"))
	suiteState.Require().NoError(err)
	suiteState.Equal("79fda9a16d0e718677cd8a1c01eb859dc90e18adf7fcd2a7c18cfb8819ab4903  istio-1.31.0-linux-amd64.tar.gz\n1ebedd17ae4d49c56887d0d7c264486d6cf2d0e43d5fefcad78afdf922fee112  istio-1.31.0-linux-arm64.tar.gz\nb5f07c7ea1a3a9b805ff8690c3a591621feef8b714d346a2e240c898cd2a84b0  istio-1.31.0-linux-armv7.tar.gz\nafff9ca5a07c7ea3aad81276ee5eb47b5f87bc386f5dae6eb54c57e50629ab33  istio-1.31.0-osx-amd64.tar.gz\n263844544567c2137576a24c19c087a48f3b9b8188d76176efdd5eacc23f553c  istio-1.31.0-osx-arm64.tar.gz\n", string(data))
	data, err = os.ReadFile(workflowBuildPath(suiteState.T(), "../evals/README.md"))
	suiteState.Require().NoError(err)
	for _, expected := range []string{"v0.0.22 checker and standalone agent", "in-process", "checker-only", "not a signature or attestation", "not a sandbox", "efd3cf05f607ab7804c4a16b543d916ebfc51a74", "published C3 source", "601554bed0375083b88d62d115d3ed0cde324fed", "4942815c1af74894f58640938c12c4f84623ec69", "historical local checks", "published H4", "b918eee433d300093cc73b72f05d9a8e55a2a88e", "Published C4", "4452cb14c4fb6767ee4c02ccbae7f38ab3db8e7e", "published H5", "aa2cde6a978111cd162b79082084175bf5b0e820", "Published C5", "ec4b1c715e7cbbfbd568a99d3c27798952cb3550", "not yet committed or validated", "No new external", "SECURITY clearance", "Operator-only Kiali multicluster", "Automatic protected evaluation, reporter mapping, and scheduled baseline branches for Kiali multicluster are deferred", "0.8", "mcpchecker-reviewed", "b6fb5348dd66280245e3d98d8d93f416ece453c3", "0c7171a9fdc4762366a78e9a0e3f226ff6e6bbd3"} {
		suiteState.Contains(strings.Join(strings.Fields(strings.ReplaceAll(string(data), "**", "")), " "), expected)
	}
}
