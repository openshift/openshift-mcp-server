// Shared downstream presentation. runtime.js owns MCP messaging, data
// normalization, quantity comparisons, sorting, and resize reporting.
import React, { useState } from "react";
import { createRoot } from "react-dom/client";
import { flushSync } from "react-dom";
// The package-wide barrel imports styles for unused components too. Import
// component families directly, preserving all their interactive-state styles.
import { Alert } from "@patternfly/react-core/dist/esm/components/Alert/index.js";
import { Card, CardBody } from "@patternfly/react-core/dist/esm/components/Card/index.js";
import { CodeBlock, CodeBlockCode } from "@patternfly/react-core/dist/esm/components/CodeBlock/index.js";
import { DescriptionList, DescriptionListDescription, DescriptionListGroup,
  DescriptionListTerm } from "@patternfly/react-core/dist/esm/components/DescriptionList/index.js";
import { ExpandableSection } from "@patternfly/react-core/dist/esm/components/ExpandableSection/index.js";
import { Label } from "@patternfly/react-core/dist/esm/components/Label/index.js";
import { Skeleton } from "@patternfly/react-core/dist/esm/components/Skeleton/index.js";
import { Title } from "@patternfly/react-core/dist/esm/components/Title/index.js";
import { Table, Tbody, Td, Th, Thead, Tr } from "@patternfly/react-table";
import "@patternfly/react-core/dist/styles/base.css";
import "@rhds/tokens/css/global.css";
import logoLight from "./assets/red-hat-logo-light.svg";
import logoDark from "./assets/red-hat-logo-dark.svg";
import "./styles.css";

const rootElement = document.getElementById("app");
rootElement.removeAttribute("role");
rootElement.className = "";
const root = createRoot(rootElement);
const appName = document.body.dataset.appName || "Kubernetes data";
const asText = value => value == null ? "" : typeof value === "object" ? JSON.stringify(value) : String(value);

function Namespace({ value }) {
  return <span className="namespace"><Label color="purple" isCompact>NS</Label><span>{asText(value)}</span></span>;
}

function Status({ value }) {
  const label = asText(value);
  const color = /^(active|running|ready|true|succeeded|complete)$/i.test(label) ? "green"
    : /error|fail|backoff|terminat/i.test(label) ? "red"
    : /pending|unknown|notready|false/i.test(label) ? "orange" : "grey";
  return label ? <Label color={color} isCompact>{label}</Label> : null;
}

function Cell({ column, value }) {
  if (column === "Namespace" && value) return <Namespace value={value} />;
  if (column === "Status") return <Status value={value} />;
  return <span className={column === "Name" ? "resource-name" : undefined}>{asText(value)}</span>;
}

function DataTable({ columns, rows, sortColumn, descending, onSort, label = "Resource data" }) {
  return <div className="table-scroll" tabIndex={0} role="region" aria-label={label + " (scroll for more columns)"}>
    <Table aria-label={label} variant="compact" gridBreakPoint="">
      <Thead><Tr>{columns.map((column, index) => <Th key={column} scope="col"
        sort={onSort ? { sortBy: { index: columns.indexOf(sortColumn), direction: descending ? "desc" : "asc" },
          columnIndex: index, onSort: () => onSort(column) } : undefined}>{column}</Th>)}</Tr></Thead>
      <Tbody>{rows.map((row, i) => <Tr key={i}>{columns.map(column => <Td key={column} dataLabel={column}>
        <Cell column={column} value={row[column]} />
      </Td>)}</Tr>)}</Tbody>
    </Table>
  </div>;
}

function Fields({ value }) {
  if (value && typeof value === "object" && Object.keys(value).length) {
    if (Array.isArray(value)) return <ol>{value.map((item, index) => <li key={index}><Fields value={item} /></li>)}</ol>;
    return <DescriptionList isCompact orientation={{ sm: "horizontal" }}>{Object.entries(value).map(([key, item]) =>
      <DescriptionListGroup key={key}><DescriptionListTerm>{key}</DescriptionListTerm>
        <DescriptionListDescription>{key === "Namespace" ? <Namespace value={item} /> : <Fields value={item} />}</DescriptionListDescription>
      </DescriptionListGroup>)}</DescriptionList>;
  }
  return <span>{value && typeof value === "object" ? "None" : asText(value) || "—"}</span>;
}

function Disclosure({ name, children }) {
  const [expanded, setExpanded] = useState(false);
  return <ExpandableSection toggleText={name} isExpanded={expanded} onToggle={(_, open) => setExpanded(open)}>{children}</ExpandableSection>;
}

function Section({ name, children }) {
  return <section aria-label={name}><Title headingLevel="h2" size="md">{name}</Title>{children}</section>;
}

function ResourceView({ data }) {
  const metadata = data.metadata || {};
  const isPod = data.kind === "Pod" && data.apiVersion === "v1";
  const overview = { Name: metadata.name, Kind: data.kind, "API version": data.apiVersion };
  if (metadata.namespace) overview.Namespace = metadata.namespace;
  if (metadata.creationTimestamp) overview.Created = metadata.creationTimestamp;
  if (data.status?.phase) overview.Status = data.status.phase;
  if (isPod && data.spec?.nodeName) overview.Node = data.spec.nodeName;
  if (isPod && data.status?.podIP) overview["Pod IP"] = data.status.podIP;
  const readiness = isPod && Array.isArray(data.status?.conditions) ? data.status.conditions.find(condition => condition.type === "Ready") : undefined;
  return <div className="resource-details">
    {readiness && <Alert variant={readiness.status === "True" ? "success" : "warning"} isInline
      title={readiness.status === "True" ? "Pod is ready" : "Pod is not ready"}>{readiness.message || readiness.reason}</Alert>}
    <Section name="Overview"><Fields value={overview} /></Section>
    {metadata.labels && <Section name="Labels"><Fields value={metadata.labels} /></Section>}
    {metadata.annotations && <Disclosure name="Annotations"><Fields value={metadata.annotations} /></Disclosure>}
    {isPod && [["Containers", "containers", "containerStatuses"], ["Init containers", "initContainers", "initContainerStatuses"],
      ["Ephemeral containers", "ephemeralContainers", "ephemeralContainerStatuses"]].map(([name, specKey, statusKey]) => {
      const containers = data.spec?.[specKey];
      if (!Array.isArray(containers) || !containers.length) return null;
      const statuses = Array.isArray(data.status?.[statusKey]) ? data.status[statusKey] : [];
      const rows = containers.map(container => {
        const status = statuses.find(candidate => candidate?.name === container.name);
        const state = status?.state || {}, stateName = Object.keys(state)[0], detail = state[stateName] || {};
        return { Name: container.name, Image: container.image, Ready: status ? status.ready ? "Yes" : "No" : "Unknown",
          Restarts: status?.restartCount ?? "—", State: detail.reason || stateName || "Unknown", Message: detail.message || "" };
      });
      return <Section name={name} key={name}><DataTable label={name} columns={["Name", "Image", "Ready", "Restarts", "State", "Message"]} rows={rows} /></Section>;
    })}
    {Array.isArray(data.status?.conditions) && data.status.conditions.length > 0 && <Section name="Conditions"><DataTable label="Conditions"
      columns={["Type", "Status", "Reason", "Message", "Last transition"]}
      rows={data.status.conditions.map(condition => ({ Type: condition.type, Status: condition.status, Reason: condition.reason,
        Message: condition.message, "Last transition": condition.lastTransitionTime }))} /></Section>}
    {data.spec && <Disclosure name="Specification"><Fields value={data.spec} /></Disclosure>}
    {data.status && <Disclosure name="Status details"><Fields value={data.status} /></Disclosure>}
    <Disclosure name="Raw resource"><CodeBlock><CodeBlockCode>{JSON.stringify(data, null, 2)}</CodeBlockCode></CodeBlock></Disclosure>
  </div>;
}

function ContentSkeleton({ kind }) {
  if (kind === "resource") {
    return <div className="loading loading-resource" aria-hidden="true">
      <Skeleton width="85%" height="24px" />
      <div className="loading-fields">{[0, 1, 2, 3].map(row => <React.Fragment key={row}>
        <Skeleton width="75%" /><Skeleton width="90%" />
      </React.Fragment>)}</div>
    </div>;
  }
  return <div className="loading loading-table" aria-hidden="true">{[0, 1, 2].map(row =>
    <div className="loading-row" key={row}><Skeleton width="90%" /><Skeleton width="70%" /><Skeleton width="60%" /></div>
  )}</div>;
}

function Shell({ model }) {
  const title = model.type === "resource" ? [model.data.kind, model.data.metadata?.name].filter(Boolean).join(" ") || appName : appName;
  const hasRows = model.type === "table" && model.rows.length > 0;
  const kind = document.body.dataset.appKind;
  const loading = model.type === "loading";
  let description = kind === "metrics" ? "Current CPU and memory usage. No historical trend is available."
    : model.type === "resource" ? "Object identity, state, and source data" : "Kubernetes resource inventory";
  if (model.type === "table") description = `${model.rows.length} ${model.rows.length === 1 ? "row" : "rows"} returned · ${description}`;
  const announcement = loading ? "Loading data…" : model.type === "status" ? model.message : "";
  return <main className="openshift-mcp-app" aria-label="OpenShift MCP App">
    <Card component="article" aria-labelledby="view-title" className="result-card" aria-busy={loading || undefined}>
      <CardBody>
        <header className="identity"><div>
          <img className="logo logo-light" src={logoLight} alt="Red Hat" />
          <img className="logo logo-dark" src={logoDark} alt="Red Hat" />
        </div></header>
        <header className="task-heading"><Title id="view-title" headingLevel="h1" size="lg" className={loading ? "pf-v6-screen-reader" : undefined}>{title}</Title>
          {loading ? <>
            <Skeleton className="title-skeleton" width="75%" aria-hidden="true" />
            <Skeleton className="description-skeleton" width="90%" aria-hidden="true" />
          </> : <p className="result-description">{description}</p>}
        </header>
        <div className="result-content">{loading ? <ContentSkeleton kind={kind} />
          : model.type === "status" ? <p className={model.error ? "error-state" : "empty-state"}>{model.message}</p>
          : model.type === "resource" ? <ResourceView key={title} data={model.data} />
          : <DataTable {...model} label={appName} />}</div>
        {hasRows && <footer className="result-footer">Sorting applies to the returned rows. Scroll horizontally for additional fields.</footer>}
      </CardBody>
    </Card>
    <span className="pf-v6-screen-reader" role="status" aria-live="polite" aria-atomic="true">{announcement}</span>
  </main>;
}

function render(model) {
  // Keep the runtime's synchronous error recovery and ResizeObserver behavior.
  flushSync(() => root.render(<Shell model={model} />));
}

function theme(value) {
  document.documentElement.classList.toggle("pf-v6-theme-dark", value === "dark");
}
const systemTheme = matchMedia("(prefers-color-scheme: dark)");
theme(systemTheme.matches ? "dark" : "light");
systemTheme.addEventListener("change", () => {
  if (!document.documentElement.dataset.theme) theme(systemTheme.matches ? "dark" : "light");
});

window.mcpAppPresentation = {
  table: model => render({ ...model, type: "table" }),
  resource: data => {
    // Reject malformed input before entering React, so runtime.js can display
    // the existing diagnostic and recover on the next valid result.
    if (Array.isArray(data.status?.conditions) && data.status.conditions.some(item => !item || typeof item !== "object")) throw new Error("invalid condition");
    if (data.kind === "Pod" && data.apiVersion === "v1") {
      for (const key of ["containers", "initContainers", "ephemeralContainers"]) {
        if (Array.isArray(data.spec?.[key]) && data.spec[key].some(item => !item || typeof item !== "object")) throw new Error("invalid container");
      }
    }
    render({ type: "resource", data });
  },
  status: (message, error = false) => render({ type: "status", message, error }),
  theme,
};
render({ type: "loading" });
