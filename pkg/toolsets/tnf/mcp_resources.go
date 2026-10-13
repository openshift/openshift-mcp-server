package tnf

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

func initMCPResources() []api.ServerResource {
	return []api.ServerResource{
		{
			Resource: api.Resource{
				URI:         "tnf://domain-knowledge/fencing",
				Name:        "tnf-fencing-domain-knowledge",
				Description: "TNF two-node fencing domain knowledge: quorum rules, split-brain risk matrix, and recovery procedures",
				MIMEType:    "text/markdown",
			},
			Handler: resourceFencingDomainKnowledge,
		},
	}
}

func resourceFencingDomainKnowledge(_ context.Context) (*api.ResourceContent, error) {
	return &api.ResourceContent{Text: fencingDomainKnowledge}, nil
}

const fencingDomainKnowledge = `# TNF Fencing Domain Knowledge

## How Two-Node Fencing Works

TNF clusters use **pacemaker** and **corosync** for high availability:

1. **Corosync** provides cluster communication and membership between the 2 nodes
2. **Pacemaker** manages cluster resources (services, virtual IPs) and decides placement
3. **STONITH** (Shoot The Other Node In The Head) is the fencing mechanism:
   - Each node has a fence device (IPMI via fence_ipmilan or Redfish via fence_redfish) that targets the OTHER node
   - When a node fails or loses communication, the surviving node uses the fence device to power-cycle the failed node via its BMC
   - This prevents **split-brain**: a condition where both nodes think they own shared resources, causing data corruption

**Two-node quorum** is special:
- Normal quorum requires a majority (>50%) of votes. With 2 nodes, losing 1 node means only 50% — no majority
- Corosync uses the two_node: 1 and wait_for_all: 1 settings to handle this
- wait_for_all: Both nodes must be present at initial startup before quorum is granted
- Once running, if one node fails, the remaining node retains quorum (provided it successfully fences the other)
- The no-quorum-policy property controls what happens when quorum is lost (usually "stop" — stop all resources)

## Split-Brain Risk Assessment

Analyze collected data for these risk conditions:

| Condition | Risk Level | Action Required |
|-----------|-----------|-----------------|
| STONITH disabled | **CRITICAL** | Re-enable immediately — cluster has no fencing protection |
| Fence device stopped/failed | **HIGH** | Fencing will fail on demand — check BMC connectivity |
| BMC credentials missing | **HIGH** | Fence agent cannot authenticate to BMC |
| BMC address not configured | **HIGH** | Fence agent has no target to fence |
| One node offline (pacemaker) | **MEDIUM** | Check if fencing occurred — verify other node has quorum |
| Quorum lost | **HIGH** | Resources may be stopped depending on no-quorum-policy |
| Both nodes online, healthy | **LOW** | Normal operation |

## Common Issues and Recovery Procedures

### 1. STONITH Disabled

- **Symptom:** stonith-enabled: false in pcs property list
- **Risk:** No fencing protection — split-brain can occur
- **Recovery:** Verify fence devices are configured and working, then run: pcs property set stonith-enabled=true
- **Verify:** pcs stonith status shows devices Started

### 2. BMC Unreachable / Credentials Invalid

- **Symptom:** Fence device fails to operate; BMH shows credential errors
- **Risk:** Fencing will fail when needed — node cannot be power-cycled
- **Recovery:** Check BMC network connectivity, update the credential Secret referenced by the BareMetalHost
- **Verify:** fence_<agent> -a <bmc_ip> -l <user> -p <pass> -o status (from the node)

### 3. Node Won't Rejoin After Fencing

- **Symptom:** Node was fenced but pacemaker shows it OFFLINE
- **Recovery:** On the fenced node after it boots: check pcs status, clear stale fencing state with pcs stonith history cleanup <node>, then pcs cluster start if needed
- **Check:** BMC power state via fence_<agent> -o status, verify node booted successfully

### 4. Quorum Lost (Single Node Running)

- **Symptom:** Quorate: No in corosync-quorumtool; resources may be stopped
- **Context:** With no-quorum-policy: stop, pacemaker stops all resources when quorum is lost
- **Recovery:** If the other node is truly down and fenced, the remaining node can be given quorum with pcs quorum unblock or by using --force flags (use with caution)

### 5. Fence Device Misconfigured

- **Symptom:** Wrong agent type, incorrect BMC IP, or wrong parameters
- **Recovery:** pcs stonith update <device_name> <param>=<value> to correct parameters
- **Verify:** pcs stonith config to review, fence_<agent> -o status to test

### 6. Fence Race (Both Nodes Fencing Each Other)

- **Symptom:** Both nodes tried to fence each other simultaneously
- **Diagnosis:** Check pcs stonith history on both nodes — one should have won
- **Recovery:** The node that lost the fence race should have been power-cycled. If both are up, check which one has quorum and resources
`
