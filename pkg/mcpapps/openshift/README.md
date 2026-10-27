# OpenShift MCP Apps presentation

Shared PatternFly presentation for downstream toolsets. It uses upstream's
public MCP Apps API while keeping the generic upstream presentation unchanged.

## Ownership and selection

`pkg/mcpapps` remains the unchanged upstream implementation. This package owns
the OpenShift HTML, runtime, React/PatternFly components, styles, fonts, and
branding. Its `Table`, `Metrics`, and `Resource` constructors use upstream's
public `Custom()` API; no MCP server registration changes are needed.

`pkg/toolsets/core/app_overrides.go` selects this presentation for the existing
namespace/project, Pod, resource, event, and metrics apps. It preserves their
URIs, names, descriptions, and resource metadata. The tool definitions,
handlers, permissions, and structured results are unchanged. Other toolsets
continue to use their existing apps. To remove the downstream presentation,
remove that selection call from the core toolset.

The common style lives in `frontend/styles.css`, and the shared PatternFly
components live in `frontend/main.jsx`; tools do not need separate styles.
`runtime.js` is a downstream-owned adaptation of upstream's standard runtime
for lifecycle handling, normalization, sorting, and resize notifications.
Review upstream runtime fixes here too; the isolation does not automatically
synchronize the two implementations.

## Building and validation

```sh
make patternfly-assets
make build
```

This installs pinned frontend dependencies, bundles `presentation.js` and
`presentation.css`, and runs the normal server build. After frontend changes,
regenerate and commit both bundles with their sources. `make build` and
container builds embed those bundles directly, without requiring Node.js.
The existing `browser-tests.yaml` workflow runs both upstream and downstream
browser tests through the shared `make browser-test` target. Its downstream
prerequisite, `make patternfly-assets-check`, rebuilds assets in memory and
compares them with the committed bundles without overwriting them. A stale or
missing bundle fails the check; run `make patternfly-assets` and commit the
regenerated files to resolve it. Browser tests then exercise the checked bundles.

For Go-only changes, use `make build`. At runtime there is no frontend server,
Node.js dependency, CDN, or external font/image request.

HTML is assembled when a client reads an app resource, so tool discovery and
configuration reloads do not retain a full document per tool. Embedded assets
are shared across apps. The frontend imports PatternFly component families
directly to avoid bundling unrelated component styles; it retains their full
state styles and the shared base styles, including embedded fonts.

Keep the existing client configuration and `apps_enabled = true`. Restart its
MCP server after rebuilding. Try `namespaces_list`, `pods_list`, `pods_get`,
`resources_get`, `pods_top`, and `nodes_top`. Metrics require the Metrics API.

`test/browser/extensions/openshift/patternfly.spec.mjs` checks the actual
downstream bundle. The shared `make browser-test` target automatically discovers
it alongside the upstream tests. The downstream Makefile adds the frontend
asset-check prerequisite; the server is built after those assets are verified.
The full target still requires a cluster, as it runs the basic-host tests too.


## Visual reference

The local `patternfly-mcp-app-shell` demo supplies the visual reference and
approved Red Hat artwork. The card layout follows its `affeb41` revision:
in-card branding, a rounded title accent, descriptions instead of an eyebrow,
and content-specific loading placeholders. Its fixture data, simulated host,
refresh controls, and no-op actions are not included.

The iframe document and root wrappers stay transparent; the card keeps its
PatternFly surface. The opt-in `pf-mcp-app-document` class clears page colors
and background images. The runtime applies host theme context to the document.
The host also needs to match its iframe element's color scheme to the app's
scheme so browsers can show the host background around the rounded card.

Metrics remain tables of the returned CPU and memory quantities. The current
tool data does not supply the used/total measurement required by the demo's
node-memory percentage charts. This presentation adds no Kubernetes actions,
new metrics, or tool capabilities.
