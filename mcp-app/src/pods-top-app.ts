import { App } from "@modelcontextprotocol/ext-apps";

interface ContainerMetric {
  name: string;
  cpuMillicores: number;
  memoryBytes: number;
}

interface PodMetric {
  name: string;
  namespace: string;
  containers: ContainerMetric[];
  totalCpuMillicores: number;
  totalMemoryBytes: number;
}

interface PodsTopData {
  pods: PodMetric[];
}

type SortField = "name" | "namespace" | "cpu" | "memory";
type SortDir = "asc" | "desc";

const appContainer = document.getElementById("app")!;
const app = new App({ name: "Pods Resource Usage", version: "1.0.0" });

let currentData: PodsTopData | null = null;
let sortField: SortField = "cpu";
let sortDir: SortDir = "desc";
let refreshing = false;

function formatCPU(millis: number): string {
  if (millis >= 1000) {
    return `${(millis / 1000).toFixed(1)} cores`;
  }
  return `${millis}m`;
}

function formatMemory(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} Gi`;
  }
  if (bytes >= 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(0)} Mi`;
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(0)} Ki`;
  }
  return `${bytes} B`;
}

function sortPods(pods: PodMetric[]): PodMetric[] {
  const sorted = [...pods];
  const mul = sortDir === "asc" ? 1 : -1;
  sorted.sort((a, b) => {
    switch (sortField) {
      case "name":
        return mul * a.name.localeCompare(b.name);
      case "namespace":
        return mul * a.namespace.localeCompare(b.namespace);
      case "cpu":
        return mul * (a.totalCpuMillicores - b.totalCpuMillicores);
      case "memory":
        return mul * (a.totalMemoryBytes - b.totalMemoryBytes);
    }
  });
  return sorted;
}

function sortArrow(field: SortField): string {
  if (sortField !== field) return "";
  return `<span class="sort-arrow">${sortDir === "asc" ? "\u25B2" : "\u25BC"}</span>`;
}

function renderDashboard(data: PodsTopData) {
  const pods = sortPods(data.pods);
  const maxCPU = Math.max(...pods.map((p) => p.totalCpuMillicores), 1);
  const maxMem = Math.max(...pods.map((p) => p.totalMemoryBytes), 1);

  appContainer.innerHTML = `
    <div class="header">
      <div>
        <h1>Pods Resource Usage</h1>
        <span class="pod-count">${pods.length} pod${pods.length !== 1 ? "s" : ""}</span>
      </div>
      <button class="refresh-btn" id="refresh-btn" ${refreshing ? "disabled" : ""}>
        ${refreshing ? "Refreshing..." : "Refresh"}
      </button>
    </div>
    <div class="legend">
      <div class="legend-item"><div class="legend-dot cpu"></div>CPU</div>
      <div class="legend-item"><div class="legend-dot mem"></div>Memory</div>
    </div>
    ${
      pods.length === 0
        ? '<div class="empty">No pod metrics available</div>'
        : `<table>
      <thead>
        <tr>
          <th data-sort="name">Pod${sortArrow("name")}</th>
          <th data-sort="namespace">Namespace${sortArrow("namespace")}</th>
          <th data-sort="cpu" class="metric-cell">CPU${sortArrow("cpu")}</th>
          <th data-sort="memory" class="metric-cell">Memory${sortArrow("memory")}</th>
        </tr>
      </thead>
      <tbody>
        ${pods
          .map(
            (pod) => `
          <tr title="${pod.name}&#10;${pod.containers.map((c) => `  ${c.name}: ${formatCPU(c.cpuMillicores)} / ${formatMemory(c.memoryBytes)}`).join("&#10;")}">
            <td class="name-cell">${pod.name}</td>
            <td class="ns-cell">${pod.namespace}</td>
            <td class="metric-cell">
              <div class="bar-wrap">
                <div class="bar-track">
                  <div class="bar-fill cpu" style="width: ${(pod.totalCpuMillicores / maxCPU) * 100}%"></div>
                </div>
                <span class="bar-value">${formatCPU(pod.totalCpuMillicores)}</span>
              </div>
            </td>
            <td class="metric-cell">
              <div class="bar-wrap">
                <div class="bar-track">
                  <div class="bar-fill mem" style="width: ${(pod.totalMemoryBytes / maxMem) * 100}%"></div>
                </div>
                <span class="bar-value">${formatMemory(pod.totalMemoryBytes)}</span>
              </div>
            </td>
          </tr>
        `
          )
          .join("")}
      </tbody>
    </table>`
    }
  `;

  // Attach sort handlers
  appContainer.querySelectorAll("th[data-sort]").forEach((th) => {
    th.addEventListener("click", () => {
      const field = (th as HTMLElement).dataset.sort as SortField;
      if (sortField === field) {
        sortDir = sortDir === "asc" ? "desc" : "asc";
      } else {
        sortField = field;
        sortDir = field === "name" || field === "namespace" ? "asc" : "desc";
      }
      if (currentData) renderDashboard(currentData);
    });
  });

  // Attach refresh handler
  const refreshBtn = document.getElementById("refresh-btn");
  if (refreshBtn) {
    refreshBtn.addEventListener("click", async () => {
      refreshing = true;
      if (currentData) renderDashboard(currentData);
      try {
        const result = await app.callServerTool({
          name: "pods_top",
          arguments: {},
        });
        handleResult(result);
      } catch {
        renderError("Failed to refresh pod metrics");
      } finally {
        refreshing = false;
        if (currentData) renderDashboard(currentData);
      }
    });
  }
}

function renderError(message: string) {
  appContainer.innerHTML = `<div class="error"><strong>Error:</strong> ${message}</div>`;
}

function handleResult(result: {
  content?: { type: string; text?: string }[];
  structuredContent?: Record<string, unknown>;
  isError?: boolean;
}) {
  if (result.isError) {
    const text = result.content?.find((c) => c.type === "text")?.text;
    renderError(text ?? "Tool execution failed");
    return;
  }
  if (result.structuredContent) {
    currentData = result.structuredContent as unknown as PodsTopData;
    renderDashboard(currentData);
  } else {
    const text = result.content?.find((c) => c.type === "text")?.text;
    if (text) {
      appContainer.innerHTML = `<pre style="white-space: pre-wrap; font-size: 12px;">${text}</pre>`;
    } else {
      renderError("No data received");
    }
  }
}

// Set ontoolresult BEFORE connect per MCP Apps spec
app.ontoolresult = (result) => {
  handleResult(result as Parameters<typeof handleResult>[0]);
};

// Apply theme from host context changes
app.onhostcontextchanged = (params) => {
  if (params.theme === "dark") {
    document.body.classList.add("dark");
    document.body.classList.remove("light");
  } else {
    document.body.classList.add("light");
    document.body.classList.remove("dark");
  }
};

app.connect();
