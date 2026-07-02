package fencing

import (
	"errors"
	"net"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// IsAPIUnreachable probes the Kubernetes API server via the discovery
// client. It returns true only when the API server cannot be reached
// at the network level (connection refused, no route, timeout). API-level
// errors such as Forbidden or Unauthorized indicate the server IS
// reachable and return false.
func IsAPIUnreachable(client api.KubernetesClient) bool {
	if client == nil {
		return false
	}
	_, err := client.DiscoveryClient().ServerVersion()
	if err == nil {
		return false
	}
	return isConnectionError(err)
}

func isConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection reset by peer")
}

// APIUnreachableGuide returns a structured troubleshooting guide for when
// the Kubernetes API server is unreachable. It covers three scenarios:
// failed install, upgrade/reboot, and crash/degraded.
func APIUnreachableGuide() string {
	return `# TNF Diagnostic: Kubernetes API Unreachable

The Kubernetes API server is not responding. This prevents all cluster-level
diagnostics. Use out-of-band access (BMC, SSH) to determine what happened.

---

## Determine Your Scenario

| Clue | Likely Scenario |
|------|----------------|
| Install was recently started, cluster never came up | **Scenario 1: Install failure** |
| Upgrade or MachineConfig change was in progress | **Scenario 2: Upgrade/reboot** |
| Cluster was running normally, then stopped | **Scenario 3: Crash/degraded** |

---

## Common First Steps (All Scenarios)

### 1. Check Node Power State via BMC

Use IPMI or Redfish to verify nodes are powered on:

` + "```" + `
# IPMI
ipmitool -I lanplus -H <bmc-ip> -U <user> -P <pass> power status

# Redfish
curl -k -u <user>:<pass> https://<bmc-ip>/redfish/v1/Systems/1 | jq '.PowerState'
` + "```" + `

### 2. SSH to Nodes

If nodes booted far enough, SSH access may be available:

` + "```" + `
ssh core@<node-ip>
` + "```" + `

### 3. Check API VIP Assignment

From any node, verify the API VIP is assigned to a network interface:

` + "```" + `
ip addr show | grep <api-vip>
` + "```" + `

If the VIP is not assigned, the API server or keepalived is not running.

### 4. Check DNS Resolution

` + "```" + `
dig api.<cluster-name>.<base-domain>
` + "```" + `

---

## Scenario 1: Failed or In-Progress Install

The cluster has never fully bootstrapped. The assisted-installer agent
or bootstrap process may still be running or may have failed.

### Check Agent-Based Installer Status

` + "```" + `
# Agent service logs (ABI installs)
journalctl -u agent --no-pager -n 100

# Assisted-installer local API (if agent is running)
curl -s http://localhost:8090/api/assisted-install/v2/clusters | jq '.[].status'
` + "```" + `

### Check Bootstrap Progression

` + "```" + `
# Has bootstrap completed?
ls -la /opt/openshift/.bootkube.done

# Are static pod manifests present?
ls /etc/kubernetes/manifests/

# Is kubelet running?
systemctl status kubelet

# etcd status
sudo crictl ps --name etcd
sudo crictl logs $(sudo crictl ps --name etcd -q) 2>&1 | tail -20
` + "```" + `

### Common Install Failures

- **DNS misconfiguration** — api/api-int/*.apps must resolve correctly
- **Pull secret invalid** — check /root/.docker/config.json or /var/lib/kubelet/config.json
- **BMC unreachable during install** — needed for host provisioning
- **Certificate issues** — clock skew between nodes causes TLS failures
- **Disk/storage** — insufficient space on /var or /sysroot

---

## Scenario 2: Upgrade or Reboot in Progress

A node rebooted for an OS update (MCO) or configuration change. The API
should return within 5-15 minutes on a two-node cluster.

### Check OS Update Status

` + "```" + `
# Current and pending OS deployments
rpm-ostree status

# Was a reboot triggered by MCO?
journalctl -u machine-config-daemon --no-pager -n 50
` + "```" + `

### Check Kubelet Recovery

` + "```" + `
systemctl status kubelet
journalctl -u kubelet --no-pager -n 50
` + "```" + `

### Two-Node Etcd Considerations

In a 2-node cluster, if one node reboots, etcd loses quorum until it
returns. The API will be unavailable until both etcd members are healthy:

` + "```" + `
sudo crictl ps --name etcd
sudo crictl exec $(sudo crictl ps --name etcd -q) etcdctl endpoint health \
  --cluster --cacert /etc/kubernetes/static-pod-certs/configmaps/etcd-serving-ca/ca-bundle.crt \
  --cert /etc/kubernetes/static-pod-certs/secrets/etcd-all-certs/etcd-serving-master-0.crt \
  --key /etc/kubernetes/static-pod-certs/secrets/etcd-all-certs/etcd-serving-master-0.key
` + "```" + `

### If the Upgrade Appears Stuck

- Check if the node completed its reboot: ` + "`uptime`" + `
- Check if the other node is still running and healthy
- Check pacemaker: has fencing been triggered during the upgrade?

---

## Scenario 3: Crash or Degraded State

The cluster was running and stopped unexpectedly. A node may have crashed,
or a critical service may have failed.

### Check What Happened

` + "```" + `
# Recent kubelet activity
journalctl -u kubelet --since "30 min ago" --no-pager

# Recent system events
journalctl --since "30 min ago" --priority err --no-pager

# Are control-plane containers running?
sudo crictl ps --name kube-apiserver
sudo crictl ps --name etcd
sudo crictl ps --name kube-controller-manager
` + "```" + `

### Check Pacemaker and Fencing

On a TNF cluster, the surviving node may have fenced the failed node:

` + "```" + `
pcs status
pcs stonith history
` + "```" + `

If fencing occurred:
- Identify which node was fenced and which survived
- Check BMC power state of the fenced node
- The fenced node may need manual power-on and cluster rejoin

### Etcd Recovery

` + "```" + `
# Check etcd member list and health
sudo crictl exec $(sudo crictl ps --name etcd -q) etcdctl member list \
  --cacert /etc/kubernetes/static-pod-certs/configmaps/etcd-serving-ca/ca-bundle.crt \
  --cert /etc/kubernetes/static-pod-certs/secrets/etcd-all-certs/etcd-serving-master-0.crt \
  --key /etc/kubernetes/static-pod-certs/secrets/etcd-all-certs/etcd-serving-master-0.key
` + "```" + `

### If Both Nodes Are Down

This is a critical situation. Use BMC to:
1. Check power state of both nodes
2. Power on nodes via IPMI/Redfish
3. Access BMC virtual console to observe boot process
4. Once nodes boot, follow Scenario 1 or 2 steps depending on whether
   the cluster was previously installed

---

## Next Steps

Once the API server is reachable again, re-run the TNF diagnostic tools:
- ` + "`tnf_check_fencing_config`" + ` — validate fencing configuration
- ` + "`tnf_check_stonith_status`" + ` — check pacemaker and STONITH state
- ` + "`tnf-troubleshoot`" + ` — full troubleshooting guide with live data
`
}
