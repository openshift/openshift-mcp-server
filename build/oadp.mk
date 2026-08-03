# OADP / Velero installation for local development and evals
#
# Installs Velero with MinIO as the object storage backend in the openshift-adp
# namespace. Also applies the OADP CRDs (DPA, DataProtectionTest) and creates a
# sample DataProtectionApplication so that all OADP evals can run on a Kind cluster.

VELERO_VERSION ?= v1.16.2
VELERO_PLUGIN_AWS_VERSION ?= v1.12.2
VELERO_CLI = $(shell pwd)/_output/tools/bin/velero
VELERO_OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')
VELERO_ARCH = $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
OADP_NAMESPACE = openshift-adp

VELERO_RELEASE_URL = https://github.com/vmware-tanzu/velero/releases/download/$(VELERO_VERSION)
OADP_CRD_BASE_URL = https://raw.githubusercontent.com/openshift/oadp-operator/oadp-1.5/config/crd/bases

define MINIO_DEPLOYMENT
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: $(OADP_NAMESPACE)
  name: minio
  labels:
    component: minio
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      component: minio
  template:
    metadata:
      labels:
        component: minio
    spec:
      volumes:
      - name: storage
        emptyDir: {}
      containers:
      - name: minio
        image: quay.io/minio/minio:latest
        imagePullPolicy: IfNotPresent
        args:
        - server
        - /storage
        env:
        - name: MINIO_ACCESS_KEY
          value: "minio"
        - name: MINIO_SECRET_KEY
          value: "minio123"
        ports:
        - containerPort: 9000
        volumeMounts:
        - name: storage
          mountPath: "/storage"
---
apiVersion: v1
kind: Service
metadata:
  namespace: $(OADP_NAMESPACE)
  name: minio
  labels:
    component: minio
spec:
  type: ClusterIP
  ports:
    - port: 9000
      targetPort: 9000
      protocol: TCP
  selector:
    component: minio
endef
export MINIO_DEPLOYMENT

define SAMPLE_DPA
apiVersion: oadp.openshift.io/v1alpha1
kind: DataProtectionApplication
metadata:
  name: velero-sample
  namespace: $(OADP_NAMESPACE)
spec:
  configuration:
    velero:
      defaultPlugins:
        - aws
  backupLocations:
    - velero:
        provider: aws
        default: true
        objectStorage:
          bucket: velero
        config:
          region: minio
          s3ForcePathStyle: "true"
          s3Url: http://minio.$(OADP_NAMESPACE).svc:9000
        credential:
          name: cloud-credentials
          key: cloud
endef
export SAMPLE_DPA

##@ OADP / Velero

.PHONY: velero-cli
velero-cli: ## Download the Velero CLI
	@if [ -f $(VELERO_CLI) ]; then \
		echo "Velero CLI already installed at $(VELERO_CLI)"; \
	else \
		set -e; \
		echo "Downloading Velero CLI $(VELERO_VERSION)..."; \
		mkdir -p $(shell dirname $(VELERO_CLI)); \
		TMPDIR=$$(mktemp -d); \
		curl -fsSL $(VELERO_RELEASE_URL)/velero-$(VELERO_VERSION)-$(VELERO_OS)-$(VELERO_ARCH).tar.gz -o $$TMPDIR/velero.tar.gz; \
		tar -xzf $$TMPDIR/velero.tar.gz -C $$TMPDIR; \
		cp $$TMPDIR/velero-$(VELERO_VERSION)-$(VELERO_OS)-$(VELERO_ARCH)/velero $(VELERO_CLI); \
		chmod +x $(VELERO_CLI); \
		rm -rf $$TMPDIR; \
		echo "Velero CLI installed at $(VELERO_CLI)"; \
	fi

.PHONY: oadp-install
oadp-install: velero-cli ## Install Velero + MinIO for OADP eval testing
	@echo "========================================="
	@echo "Installing Velero $(VELERO_VERSION) with MinIO"
	@echo "for OADP eval testing"
	@echo "========================================="
	@echo ""
	@echo "Creating namespace $(OADP_NAMESPACE)..."
	@kubectl create namespace $(OADP_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo ""
	@echo "Deploying MinIO..."
	@echo "$$MINIO_DEPLOYMENT" | kubectl apply -f -
	@echo ""
	@echo "Waiting for MinIO to be ready..."
	@kubectl wait --for=condition=available deployment/minio -n $(OADP_NAMESPACE) --timeout=5m
	@echo "✅ MinIO is ready"
	@echo ""
	@echo "Creating MinIO bucket..."
	@kubectl run minio-bucket-setup --rm -i --restart=Never --namespace $(OADP_NAMESPACE) \
		--image=quay.io/minio/mc:latest --command -- sh -c \
		"mc alias set velero http://minio:9000 minio minio123 && mc mb -p velero/velero"
	@echo "✅ MinIO bucket created"
	@echo ""
	@echo "Installing Velero..."
	@CRED_FILE=$$(mktemp); \
	printf '[default]\naws_access_key_id = minio\naws_secret_access_key = minio123\n' > $$CRED_FILE; \
	$(VELERO_CLI) install \
		--provider aws \
		--plugins velero/velero-plugin-for-aws:$(VELERO_PLUGIN_AWS_VERSION) \
		--namespace $(OADP_NAMESPACE) \
		--bucket velero \
		--secret-file $$CRED_FILE \
		--backup-location-config region=minio,s3ForcePathStyle=true,s3Url=http://minio.$(OADP_NAMESPACE).svc:9000 \
		--use-volume-snapshots=false; \
	rm -f $$CRED_FILE
	@echo ""
	@echo "Waiting for Velero deployment to be ready..."
	@kubectl wait --for=condition=available deployment/velero -n $(OADP_NAMESPACE) --timeout=5m
	@echo "✅ Velero is ready"
	@echo ""
	@echo "Installing OADP CRDs..."
	@kubectl apply -f $(OADP_CRD_BASE_URL)/oadp.openshift.io_dataprotectionapplications.yaml
	@kubectl apply -f $(OADP_CRD_BASE_URL)/oadp.openshift.io_dataprotectiontests.yaml
	@echo "✅ OADP CRDs installed"
	@echo ""
	@echo "Creating sample DataProtectionApplication..."
	@echo "$$SAMPLE_DPA" | kubectl apply -f -
	@echo "✅ Sample DPA created"
	@echo ""
	@echo "Waiting for BSL to become available..."
	@for i in $$(seq 1 60); do \
		PHASE=$$(kubectl get backupstoragelocation -n $(OADP_NAMESPACE) -o jsonpath='{.items[0].status.phase}' 2>/dev/null); \
		if [ "$$PHASE" = "Available" ]; then echo "✅ BSL is Available"; break; fi; \
		if [ $$i -eq 60 ]; then echo "⚠️  BSL not yet Available (phase: $$PHASE) — may need more time"; fi; \
		sleep 2; \
	done
	@echo ""
	@echo "========================================="
	@echo "OADP / Velero Installation Complete"
	@echo "========================================="
	@echo ""
	@echo "Velero version: $(VELERO_VERSION)"
	@echo "Namespace: $(OADP_NAMESPACE)"
	@echo ""
	@echo "Verify installation with:"
	@echo "  make oadp-status"
	@echo ""

.PHONY: oadp-uninstall
oadp-uninstall: ## Uninstall Velero and MinIO
	@echo "Uninstalling Velero and MinIO from $(OADP_NAMESPACE)..."
	@kubectl delete namespace $(OADP_NAMESPACE) --ignore-not-found
	@kubectl delete crd -l component=velero --ignore-not-found
	@kubectl delete crd dataprotectionapplications.oadp.openshift.io --ignore-not-found
	@kubectl delete crd dataprotectiontests.oadp.openshift.io --ignore-not-found
	@echo "✅ OADP / Velero uninstalled"

.PHONY: oadp-status
oadp-status: ## Show OADP / Velero status
	@echo "========================================="
	@echo "OADP / Velero Status"
	@echo "========================================="
	@echo ""
	@echo "Namespace:"
	@kubectl get namespace $(OADP_NAMESPACE) 2>/dev/null || echo "$(OADP_NAMESPACE) namespace not found"
	@echo ""
	@echo "Velero Pods:"
	@kubectl get pods -n $(OADP_NAMESPACE) 2>/dev/null || echo "No pods found"
	@echo ""
	@echo "BackupStorageLocations:"
	@kubectl get backupstoragelocation -n $(OADP_NAMESPACE) 2>/dev/null || echo "No BSLs found"
	@echo ""
	@echo "DataProtectionApplications:"
	@kubectl get dpa -n $(OADP_NAMESPACE) 2>/dev/null || echo "No DPAs found (CRD may not be installed)"
	@echo ""
	@echo "Backups:"
	@kubectl get backups.velero.io -n $(OADP_NAMESPACE) 2>/dev/null || echo "No backups found"
	@echo ""
	@echo "Restores:"
	@kubectl get restores.velero.io -n $(OADP_NAMESPACE) 2>/dev/null || echo "No restores found"
	@echo ""
	@echo "Schedules:"
	@kubectl get schedules.velero.io -n $(OADP_NAMESPACE) 2>/dev/null || echo "No schedules found"
	@echo ""
