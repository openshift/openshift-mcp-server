# NetEdge Toolset Validation Guide

This document outlines the procedure for validating the NetEdge (NIDS) toolset on an OpenShift cluster. It provides a step-by-step guide to setting up the environment, running the MCP server, and executing automated verification tasks using the Gemini agent.

## Prerequisites

- **Go**: v1.22+ (Ensure `GOROOT` is set correctly)
- **OpenShift Cluster**: Running and accessible (OCP, CRC, or similar). **Note: Evaluations WILL FAIL on Kind.**
- **oc** or **kubectl**: Installed and configured to talk to your cluster.
- **gevals**: Built and in the root directory (see Setup below).
- **Gemini CLI**: Installed (`npm install -g @google/gemini-cli`)
- **RH_GEMINI_API_KEY**: Required for the Agent.
- **OPENAI_API_KEY**: Required for the Judge (until Gemini compatibility is fully verified).

## Setup

1.  **Build/Install gevals**:
    if not already present:
    ```bash
    git clone https://github.com/mcpchecker/mcpchecker.git ../mcpchecker
    cd ../mcpchecker
    go build -o gevals ./cmd/mcpchecker
    mv gevals ../openshift-mcp-server/
    ```

2.  **Connect to OpenShift Cluster**:
    Ensure you are logged in to your OpenShift cluster and have the correct context selected.
    ```bash
    oc login ... # or export KUBECONFIG=...
    oc project default # or any namespace you have access to
    ```

## Running the Validation

1.  **Start the MCP Server**:
    Start the server in the background and redirect logs to a file.
    ```bash
    make run-server TOOLSETS=netedge MCP_LOG_LEVEL=4 MCP_LOG_FILE=server.log
    ```

2.  **Monitor Server Logs**:
    In a separate terminal, follow the server logs to see tool execution details.
    ```bash
    tail -f server.log
    ```

3.  **Run the Evaluations**:
    Run `mcpchecker` (gevals) checks using the Gemini Agent.
    
    **Test 1: Get CoreDNS Configuration**
    ```bash
    export RH_GEMINI_API_KEY=$RH_GEMINI_API_KEY && export JUDGE_API_KEY=$OPENAI_API_KEY && export JUDGE_BASE_URL="https://api.openai.com/v1" && export JUDGE_MODEL_NAME="gpt-4o" && ./gevals check evals/gemini-agent/eval.yaml --run "get-coredns-config" -v
    ```

    **Test 2: Query Prometheus Diagnostics**
    ```bash
    export RH_GEMINI_API_KEY=$RH_GEMINI_API_KEY && export JUDGE_API_KEY=$OPENAI_API_KEY && export JUDGE_BASE_URL="https://api.openai.com/v1" && export JUDGE_MODEL_NAME="gpt-4o" && ./gevals check evals/gemini-agent/eval.yaml --run "query-prometheus-ingress" -v
    ```

    **Tip**: To see the full agent conversation and debug details:
    ```bash
    ./gevals view mcpchecker-gemini-agent-netedge-eval-out.json
    ```

## Observing Results

- **Console Output**: `gevals` will show the Gemini agent's progress.
- **Server Logs**: Watch `server.log` to see the `netedge` toolset servicing requests from Gemini.
