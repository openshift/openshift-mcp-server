# ACM (Advanced Cluster Management) installation targets for OpenShift
# This file is specific to downstream OpenShift/OCP features only

.PHONY: acm-install acm-mce-install acm-operator-install acm-instance-install acm-status acm-import-cluster acm-uninstall

##@ ACM (OpenShift only)

acm-install: acm-mce-install acm-operator-install acm-instance-install ## Install MCE, ACM operator and instance

# Install MultiCluster Engine (required for ACM)
acm-mce-install:
	@echo "Installing MultiCluster Engine (MCE)..."
	@echo "Creating multicluster-engine namespace..."
	@oc create namespace multicluster-engine --dry-run=client -o yaml | oc apply -f -
	@echo "Creating MCE OperatorGroup..."
	@printf '%s\n' \
		'apiVersion: operators.coreos.com/v1' \
		'kind: OperatorGroup' \
		'metadata:' \
		'  name: multicluster-engine' \
		'  namespace: multicluster-engine' \
		'spec:' \
		'  targetNamespaces:' \
		'  - multicluster-engine' \
		| oc apply -f -
	@echo "Creating MCE Subscription..."
	@printf '%s\n' \
		'apiVersion: operators.coreos.com/v1alpha1' \
		'kind: Subscription' \
		'metadata:' \
		'  name: multicluster-engine' \
		'  namespace: multicluster-engine' \
		'spec:' \
		'  channel: stable-2.9' \
		'  name: multicluster-engine' \
		'  source: redhat-operators' \
		'  sourceNamespace: openshift-marketplace' \
		| oc apply -f -
	@echo "Waiting for MCE operator CSV to be ready..."
	@for i in {1..60}; do \
		if oc get csv -n multicluster-engine -o name 2>/dev/null | grep -q multicluster-engine; then \
			echo "MCE CSV found, waiting for Succeeded phase..."; \
			oc wait --for=jsonpath='{.status.phase}'=Succeeded csv -l operators.coreos.com/multicluster-engine.multicluster-engine -n multicluster-engine --timeout=300s && break; \
		fi; \
		echo "  Waiting for MCE CSV to appear ($$i/60)..."; \
		sleep 5; \
	done
	@echo "Creating MultiClusterEngine instance..."
	@printf '%s\n' \
		'apiVersion: multicluster.openshift.io/v1' \
		'kind: MultiClusterEngine' \
		'metadata:' \
		'  name: multiclusterengine' \
		'spec: {}' \
		| oc apply -f -
	@echo "Waiting for ManagedCluster CRD to be available..."
	@for i in $$(seq 1 120); do \
		if oc get crd managedclusters.cluster.open-cluster-management.io >/dev/null 2>&1; then \
			echo "✅ ManagedCluster CRD is now available!"; \
			break; \
		fi; \
		echo "  Waiting for ManagedCluster CRD ($$i/120)..."; \
		sleep 5; \
	done
	@echo "✓ MCE installation complete"

# Install ACM operator (Subscription, OperatorGroup, etc.)
acm-operator-install:
	@echo "Installing ACM Operator (release 2.14)..."
	oc apply -k https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14
	@echo "Waiting for ACM operator CSV to be ready..."
	@for i in $$(seq 1 60); do \
		if oc get csv -n open-cluster-management -o name 2>/dev/null | grep -q advanced-cluster-management; then \
			echo "ACM CSV found, waiting for Succeeded phase..."; \
			oc wait --for=jsonpath='{.status.phase}'=Succeeded csv -l operators.coreos.com/advanced-cluster-management.open-cluster-management -n open-cluster-management --timeout=300s && break; \
		fi; \
		echo "  Waiting for ACM CSV to appear ($$i/60)..."; \
		sleep 5; \
	done
	@echo "✓ ACM Operator installation complete"

# Install ACM instance (MultiClusterHub CR)
acm-instance-install:
	@echo "Installing ACM Instance (MultiClusterHub)..."
	@printf '%s\n' \
		'apiVersion: operator.open-cluster-management.io/v1' \
		'kind: MultiClusterHub' \
		'metadata:' \
		'  name: multiclusterhub' \
		'  namespace: open-cluster-management' \
		'spec:' \
		'  availabilityConfig: High' \
		| oc apply -f -
	@echo "Waiting for MultiClusterHub to be ready (this may take several minutes)..."
	@oc wait --for=condition=Complete --timeout=900s multiclusterhub/multiclusterhub -n open-cluster-management || true
	@echo "✓ ACM Instance installation complete"

acm-status: ## Check ACM installation status
	@echo "=========================================="
	@echo "ACM Installation Status"
	@echo "=========================================="
	@echo ""
	@echo "Namespaces:"
	@oc get namespaces | grep -E "(open-cluster-management|multicluster-engine)" || echo "No ACM namespaces found"
	@echo ""
	@echo "Operators:"
	@oc get csv -n open-cluster-management 2>/dev/null || echo "No operators found in open-cluster-management namespace"
	@echo ""
	@echo "MultiClusterHub:"
	@oc get multiclusterhub -n open-cluster-management -o wide 2>/dev/null || echo "No MultiClusterHub found"
	@echo ""
	@echo "ACM Pods:"
	@oc get pods -n open-cluster-management 2>/dev/null || echo "No pods found in open-cluster-management namespace"
	@echo ""
	@echo "ManagedClusters:"
	@oc get managedclusters 2>/dev/null || echo "No ManagedClusters found (this is normal for fresh install)"

# Import a managed cluster
# Usage: make acm-import-cluster CLUSTER_NAME=<name> MANAGED_KUBECONFIG=<path>
acm-import-cluster: ## Import a managed cluster (requires CLUSTER_NAME and MANAGED_KUBECONFIG)
	@if [ -z "$(CLUSTER_NAME)" ]; then \
		echo "Error: CLUSTER_NAME is required"; \
		echo "Usage: make acm-import-cluster CLUSTER_NAME=<name> MANAGED_KUBECONFIG=<path>"; \
		exit 1; \
	fi
	@if [ -z "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: MANAGED_KUBECONFIG is required"; \
		echo "Usage: make acm-import-cluster CLUSTER_NAME=<name> MANAGED_KUBECONFIG=<path>"; \
		exit 1; \
	fi
	@if [ ! -f "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: Kubeconfig file not found: $(MANAGED_KUBECONFIG)"; \
		exit 1; \
	fi
	@echo "==========================================="
	@echo "Importing cluster: $(CLUSTER_NAME)"
	@echo "==========================================="
	@echo "Step 1: Creating ManagedCluster resource on hub..."
	@printf '%s\n' \
		'apiVersion: cluster.open-cluster-management.io/v1' \
		'kind: ManagedCluster' \
		'metadata:' \
		'  name: $(CLUSTER_NAME)' \
		'  labels:' \
		'    cloud: auto-detect' \
		'    vendor: auto-detect' \
		'spec:' \
		'  hubAcceptsClient: true' \
		| oc apply -f -
	@echo "Step 2: Waiting for import secret to be created..."
	@for i in $$(seq 1 60); do \
		if oc get secret -n $(CLUSTER_NAME) $(CLUSTER_NAME)-import 2>/dev/null; then \
			echo "✅ Import secret created!"; \
			break; \
		fi; \
		echo "  Waiting for import secret ($$i/60)..."; \
		sleep 2; \
	done
	@echo "Step 3: Extracting import manifests..."
	@mkdir -p _output/acm-import
	@oc get secret -n $(CLUSTER_NAME) $(CLUSTER_NAME)-import -o jsonpath='{.data.crds\.yaml}' | base64 -d > _output/acm-import/$(CLUSTER_NAME)-crds.yaml
	@oc get secret -n $(CLUSTER_NAME) $(CLUSTER_NAME)-import -o jsonpath='{.data.import\.yaml}' | base64 -d > _output/acm-import/$(CLUSTER_NAME)-import.yaml
	@echo "Import manifests saved to _output/acm-import/"
	@echo "Step 4: Applying CRDs to managed cluster..."
	@KUBECONFIG=$(MANAGED_KUBECONFIG) oc apply -f _output/acm-import/$(CLUSTER_NAME)-crds.yaml
	@echo "  Waiting for CRDs to be established..."
	@sleep 5
	@echo "Step 5: Applying import manifest to managed cluster..."
	@KUBECONFIG=$(MANAGED_KUBECONFIG) oc apply -f _output/acm-import/$(CLUSTER_NAME)-import.yaml
	@echo "Step 6: Waiting for klusterlet to be ready..."
	@for i in $$(seq 1 120); do \
		if oc get managedcluster $(CLUSTER_NAME) -o jsonpath='{.status.conditions[?(@.type=="ManagedClusterConditionAvailable")].status}' 2>/dev/null | grep -q "True"; then \
			echo "✅ Cluster $(CLUSTER_NAME) is now available!"; \
			break; \
		fi; \
		echo "  Waiting for cluster to become available ($$i/120)..."; \
		sleep 5; \
	done
	@echo "==========================================="
	@echo "✓ Cluster import complete!"
	@echo "==========================================="
	@oc get managedcluster $(CLUSTER_NAME)

# Uninstall ACM (reverse order: instance first, then operator)
acm-uninstall:
	@echo "Uninstalling ACM Instance..."
	-oc delete multiclusterhub multiclusterhub -n open-cluster-management
	@echo "Waiting for MultiClusterHub to be deleted..."
	@oc wait --for=delete multiclusterhub/multiclusterhub -n open-cluster-management --timeout=300s 2>/dev/null || true
	@echo "Uninstalling ACM Operator..."
	-oc delete -k https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14
	@echo "Cleaning up namespaces..."
	-oc delete namespace open-cluster-management --timeout=300s 2>/dev/null || true
	@echo "✓ ACM uninstallation complete"

# Dump ACM manifests locally for inspection
acm-dump-manifests:
	@echo "Dumping ACM Operator manifests..."
	@mkdir -p _output/acm-manifests
	kustomize build https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14 > _output/acm-manifests/operator.yaml
	@echo "Operator manifests saved to _output/acm-manifests/operator.yaml"
	@echo ""
	@echo "Dumping ACM Instance manifests..."
	kustomize build https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/instance/base > _output/acm-manifests/instance.yaml
	@echo "Instance manifests saved to _output/acm-manifests/instance.yaml"
