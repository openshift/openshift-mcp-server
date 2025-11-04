# ACM (Advanced Cluster Management) installation targets for OpenShift
# This file is specific to downstream OpenShift/OCP features only

.PHONY: acm-install acm-mce-install acm-operator-install acm-instance-install acm-status acm-import-cluster acm-uninstall acm-dump-manifests

##@ ACM (OpenShift only)

acm-install: acm-mce-install acm-operator-install acm-instance-install ## Install MCE, ACM operator and instance

acm-mce-install: ## Install MultiCluster Engine (required for ACM)
	@./hack/acm/install-mce.sh

acm-operator-install: ## Install ACM operator
	@./hack/acm/install-operator.sh

acm-instance-install: ## Install ACM instance (MultiClusterHub CR)
	@./hack/acm/install-instance.sh

acm-status: ## Check ACM installation status
	@./hack/acm/status.sh

acm-import-cluster: ## Import a managed cluster (requires CLUSTER_NAME and MANAGED_KUBECONFIG)
	@./hack/acm/import-cluster.sh "$(CLUSTER_NAME)" "$(MANAGED_KUBECONFIG)"

acm-uninstall: ## Uninstall ACM (reverse order: instance first, then operator)
	@./hack/acm/uninstall.sh

acm-dump-manifests: ## Dump ACM manifests locally for inspection
	@echo "Dumping ACM Operator manifests..."
	@mkdir -p _output/acm-manifests
	kustomize build https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14 > _output/acm-manifests/operator.yaml
	@echo "Operator manifests saved to _output/acm-manifests/operator.yaml"
	@echo ""
	@echo "Dumping ACM Instance manifests..."
	kustomize build https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/instance/base > _output/acm-manifests/instance.yaml
	@echo "Instance manifests saved to _output/acm-manifests/instance.yaml"
