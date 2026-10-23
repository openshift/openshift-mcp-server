# Testing Patterns and Guidelines

This project follows specific testing patterns to ensure consistency, maintainability, and quality. When writing tests, adhere to the following guidelines.

## Test Prerequisites and Cache

`make test` verifies pinned Node, jq, and ShellCheck before running exactly
`go test -race -count=1 -v ./...`. The bootstrap accepts no command arguments.
Python 3.9 or newer, Go, Bash, Make, tar, sha256sum, and unzip are baseline
prerequisites. Linux x86_64 is the only supported bootstrap platform, even when
tools are already installed. All three tools must match the release and binary
checksums in `build/test-prerequisites.json`; arbitrary PATH tools are not probed
or reused. Go itself remains a trusted caller/CI compiler selected by literal
`go` on PATH, after the verified tool directories. The caller's remaining PATH,
loader options, proxy/TLS configuration, and Go environment are trusted inputs,
not a sandbox boundary for candidate Go tests. Generic CLI execution through
this helper is no longer supported.

An explicitly set `XDG_CACHE_HOME` must be nonempty,
absolute, physical (no symlink or `..` components), owned, and writable. Invalid
explicit settings fail rather than silently switching caches. Without that
setting, an unavailable or unsafe home cache (including `HOME=/`) uses a unique
owned 0700 temporary directory instead. A set `TMPDIR` must name a safe physical
writable directory; invalid settings fail. Otherwise `/tmp` is used only if it
is a safe physical writable directory; a symlinked `/tmp` requires a safe explicit
`TMPDIR` instead. Root-owned
sticky temporary directories are allowed only with a unique private child, not
a shared predictable cache. Temporary caches survive until the command finishes
and are cleaned on normal success or failure; persistent tool-cache corruption
fails closed and does not trigger fallback or an automatic redownload.

Cache path components must be owned by the caller or the system administrator
and not writable by other accounts, except for root-owned sticky temporary
ancestors. Private cache directories remain caller-owned 0700. This prevents
foreign accounts replacing cache ancestors, but does not isolate a malicious
process running as the caller or the system administrator. Downloads and cached
executables are checksum verified; this is not a universal atomic-execution or
hard wall-clock guarantee.

The offline Python regressions run through the ordinary Go build-helper suite.
They include the actual Make recipe with deliberately missing tools and an
unwritable home, isolated release-byte fixtures, cache safety and cleanup, and
fixed Go argument/environment/exit behavior. Arbitrary command arguments and
unverified PATH prerequisites are rejected or ignored before execution; pin
validation and cache-tampering regressions fail closed. These tests do not
depend on the host already having jq or ShellCheck installed.

The actual Make consumer fixtures use a test-only `runpy` platform driver and
isolated release-byte fixtures to exercise all eight command/environment/exit
and private-cache cases even on native hosts outside Linux x86_64. Native Linux
x86_64 also runs those cases without a platform override. Unsupported-platform
negative cases reject bootstrap before downloads or child execution;
on an unsupported native host that rejection is also exercised without an
override. Temporary bases are physical paths rather than assuming `/tmp` is not
a symlink. Caller-supplied command arguments, including shell metacharacters,
are rejected before preparation or child execution. These offline fixtures
do not replace official-asset CI coverage or establish macOS/ARM execution.
The capture/process-reaping tests still require Linux child-subreaper support;
portable bootstrap tests do not make the complete Go suite platform-independent.

Standard MCP app message regressions execute the actual embedded application
script through Node with an owned DOM/message-event fixture. They cover the
browser-provided parent referrer origin, unrelated source windows, parent-origin
drift, and replies targeted to the bound origin. Arbitrary cross-origin hosts
remain supported. When referrer policy hides the parent URL, the existing parent
WindowProxy identity is the trust anchor and the first valid parent JSON-RPC
message binds its origin for the session. Opaque-origin parents still require
`*` for replies and are distinguished by source window, not by the shared `null`
origin string. This does not authenticate a malicious embedding host, and these
Node fixtures are not evidence of execution in a real browser.

## Test Framework

- **Use `testify/suite`** for organizing tests into suites
- Tests should be structured using test suites that embed `suite.Suite`
- Each test file should have a corresponding suite struct (e.g., `UnstructuredSuite`, `KubevirtSuite`)
- Use the `suite.Run()` function to execute test suites

Example:
```go
type MyTestSuite struct {
    suite.Suite
}

func (s *MyTestSuite) TestSomething() {
    s.Run("descriptive scenario name", func() {
        // test implementation
    })
}

func TestMyFeature(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

## Behavior-Based Testing

- **Test the public API only** - tests should be black-box and not access internal/private functions
- **No mocks** - use real implementations and integration testing where possible
- **Behavior over implementation** - test what the code does, not how it does it
- Focus on observable behavior and outcomes rather than internal state

## Test Organization

- **Use nested subtests** with `s.Run()` to organize related test cases
- **Descriptive names** - subtest names should clearly describe the scenario being tested
- Group related scenarios together under a parent test (e.g., "edge cases", "with valid input")

Example structure:
```go
func (s *MySuite) TestFeature() {
    s.Run("valid input scenarios", func() {
        s.Run("handles simple case correctly", func() {
            // test code
        })
        s.Run("handles complex case with nested data", func() {
            // test code
        })
    })
    s.Run("edge cases", func() {
        s.Run("returns error for nil input", func() {
            // test code
        })
        s.Run("handles empty input gracefully", func() {
            // test code
        })
    })
}
```

## Assertions

- **One assertion per test case** - each `s.Run()` block should ideally test one specific behavior
- Use `testify` assertion methods: `s.Equal()`, `s.True()`, `s.False()`, `s.Nil()`, `s.NotNil()`, etc.
- Provide clear assertion messages when the failure reason might not be obvious

Example:
```go
s.Run("returns expected value", func() {
    result := functionUnderTest()
    s.Equal("expected", result, "function should return the expected string")
})
```

## Coverage

- **Aim for high test coverage** of the public API
- Add edge case tests to cover error paths and boundary conditions
- Common edge cases to consider:
  - Nil/null inputs
  - Empty strings, slices, maps
  - Negative numbers or invalid indices
  - Type mismatches
  - Malformed input (e.g., invalid paths, formats)

## Error Handling

- **Never ignore errors** in production code
- Always check and handle errors from functions that return them
- In tests, use `s.Require().NoError(err)` for operations that must succeed for the test to be valid
- Use `s.Error(err)` or `s.NoError(err)` for testing error conditions

Example:
```go
s.Run("returns error for invalid input", func() {
    result, err := functionUnderTest(invalidInput)
    s.Error(err, "expected error for invalid input")
    s.Nil(result, "result should be nil when error occurs")
})
```

## Test Helpers

- Create reusable test helpers in `internal/test/` for common testing utilities
- Test helpers should be generic and reusable across multiple test files
- Document test helpers with clear godoc comments explaining their purpose and usage

Example from this project:
```go
// FieldString retrieves a string field from an unstructured object using JSONPath-like notation.
// Examples:
//   - "spec.runStrategy"
//   - "spec.template.spec.volumes[0].containerDisk.image"
func FieldString(obj *unstructured.Unstructured, path string) string {
    // implementation
}
```

## Test Configuration and Downstream Compatibility

Downstream builds may override defaults at two layers, and tests must work
regardless of which overrides are active:

1. **Static config overrides** — `pkg/config/config_default_overrides.go`
   exposes a stub `defaultOverrides()` that downstream forks populate to
   change defaults such as `ReadOnly`, `Toolsets`, or `ToolOverrides`.
2. **Per-toolset overrides** — toolsets like `kiali` and `kubevirt` carry an
   `internal/defaults/defaults_override.go` stub that downstream uses to
   rebrand the toolset name and description (e.g. `kiali` → `ossm`).

The rules below keep the test suite portable across both layers.

### Start from `BaseDefault()`, not `New()`

`BaseMcpSuite.SetupTest()` initializes `s.Cfg` from `config.BaseDefault()`
(pure upstream defaults), then layers test-specific tweaks like
`ListOutput = "yaml"` on top. Any custom suite (`ToolsetsSuite`, etc.) must
do the same — using `config.New()` would let downstream overrides leak
into the test environment.

### Prefer merging TOML into `s.Cfg`

The dominant pattern across the test suite is to unmarshal TOML directly
into the existing `s.Cfg`. This keeps the configuration visible inline
(matching how a real config file looks) and automatically preserves the
runtime fields the suite already set (`KubeConfig`, `ListOutput`,
`ReadOnly`, etc.):

```go
s.Require().NoError(toml.Unmarshal([]byte(`
    toolsets = [ "kubevirt" ]
`), s.Cfg), "Expected to parse toolsets config")
```

See `pkg/mcp/kubevirt_test.go`, `pkg/mcp/pods_exec_test.go`, and
`pkg/mcp/helm_test.go` for representative examples.

### When `config.ReadToml` is required

`toml.Unmarshal` only does a single decode pass. Fields like
`toolset_configs` and `cluster_provider_configs` are typed as
`map[string]toml.Primitive` and need a second parse phase that uses the
TOML metadata returned by `config.ReadToml`. If your TOML uses one of those
sections, `toml.Unmarshal` will leave them unparsed and `GetToolsetConfig`
/ `GetProviderConfig` will return nothing.

In that case, use `ReadToml` and restore the runtime fields explicitly:

```go
kubeConfig := s.Cfg.KubeConfig
listOutput := s.Cfg.ListOutput
readOnly := s.Cfg.ReadOnly
cfg, err := config.ReadToml(s.T().Context(), []byte(tomlStr))
s.Require().NoError(err)
s.Cfg = cfg
s.Cfg.KubeConfig = kubeConfig
s.Cfg.ListOutput = listOutput
s.Cfg.ReadOnly = readOnly
```

`pkg/mcp/kiali_test.go` and the `TestToolHandlerReceivesToolsetConfig`
case in `pkg/mcp/mcp_config_provider_test.go` follow this pattern.

Note that the **TOML key under `[toolset_configs.<key>]` is the literal
name the toolset registers its parser under, not the toolset's exposed
name**. The Kiali toolset registers its parser as `"kiali"`
(see `pkg/kiali/config.go`), so even when downstream rebrands the
toolset to `ossm`, the section is still `[toolset_configs.kiali]`
and `params.Config.GetToolsetConfig("kiali")` is the correct lookup. Resolve
the *toolset name* dynamically via `GetName()` (see below), but keep
the *toolset_configs key* hardcoded — using the dynamic name there will
silently produce `(nil, false)`.

### Inherit runtime fields when building secondary configs

Reload tests sometimes need a separate `*config.Config` derived from
`config.New()`. This is the deliberate exception to the "start from
`BaseDefault()`" rule above: a reload simulates the SIGHUP path the
running server takes, which goes through `New()` and therefore sees
any downstream overrides. Carry across the runtime fields from `s.Cfg`
so the candidate config still points at the test kubeconfig and respects
the suite's `ReadOnly` value (see `pkg/mcp/mcp_reload_test.go`):

```go
candidateStatic := config.New()
candidateStatic.KubeConfig = s.Cfg.KubeConfig
candidateStatic.ReadOnly = s.Cfg.ReadOnly
```

### Append to `s.Cfg.Toolsets`, never reset it

Hardcoding the toolset list assumes upstream defaults and breaks downstream
builds that ship a different default set:

```go
// Good - preserves whatever the suite already has
s.Cfg.Toolsets = append(s.Cfg.Toolsets, "helm")

// Bad - assumes upstream defaults
s.Cfg.Toolsets = []string{"core", "config", "helm"}
```

### Resolve toolset names through the toolset's `GetName()`

When a test invokes a tool from a rebrandable toolset, ask the toolset for
its current name rather than hardcoding the upstream string. `kiali_test.go`
does this so the same test works whether the toolset is exposed as `kiali`
upstream or `ossm` downstream:

```go
s.toolsetName = (&kialiToolset.Toolset{}).GetName()
// ...
s.CallTool(fmt.Sprintf("%s_get_trace_details", s.toolsetName), …)
```

### Never hardcode override-prone defaults

`ReadOnly`, `Toolsets`, `ToolOverrides`, `ListOutput`, and toolset names
are all subject to downstream override. Read them from `s.Cfg` or resolve
them at runtime — never assume the upstream value.

## Examples from the Codebase

Good examples of these patterns can be found in:
- `internal/test/unstructured_test.go` - demonstrates proper use of testify/suite, nested subtests, and edge case testing
- `pkg/mcp/kubevirt_test.go` - merges TOML into `s.Cfg` for toolset selection; behavior-based MCP-layer testing
- `pkg/kubernetes/manager_test.go` - illustrates testing with proper setup/teardown and subtests
- `pkg/mcp/pods_exec_test.go` - inline TOML for `denied_resources` configuration
- `pkg/mcp/confirmation_test.go` - inline TOML for `confirmation_rules` and `confirmation_fallback`
- `pkg/mcp/helm_test.go` - appends to `s.Cfg.Toolsets` instead of resetting it
- `pkg/mcp/kiali_test.go` - resolves the toolset name dynamically via `GetName()` and uses `ReadToml` for `toolset_configs`
- `pkg/mcp/toolsets_test.go` - uses `BaseDefault()` in `ToolsetsSuite.SetupTest()` to avoid downstream config leaking
- `pkg/mcp/mcp_reload_test.go` - inherits `ReadOnly` from `s.Cfg` when building candidate configs for reload tests
- `pkg/mcp/mcp_config_provider_test.go` - demonstrates inheriting `ReadOnly` from suite config when `toolset_configs` requires `ReadToml`
- `pkg/mcp/require_tls_test.go` - shows both patterns side by side: `CoreRequireTLSSuite` merges plain `require_tls` into `s.Cfg`, while `KialiRequireTLSSuite` keeps `ReadToml` for `[toolset_configs.kiali]`
