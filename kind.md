# Kind Cluster Setup for ODH Development

This document describes how to set up a local kind (Kubernetes in Docker) cluster for developing and testing the OpenDataHub Operator.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) v0.20.0 or later
- [kubectl](https://kubernetes.io/docs/tasks/tools/) installed
- Go 1.24 or later
- At least 8GB of available RAM

## Quick Start

To create a kind cluster and run e2e tests:

```bash
# Option 1: Use the all-in-one setup command (recommended)
make kind-setup

# Option 2: Step-by-step setup
# 1. Create a kind cluster with the default configuration
make kind-create

# 2. Install OLM (Operator Lifecycle Manager)
make kind-install-olm

# 3. Install cert-manager (required for webhook certificates)
make kind-install-cert-manager

# 4. Build the operator image
make image-build IMG=local/opendatahub-operator:dev

# 5. Deploy the operator to the kind cluster (automatically creates namespace and webhook cert)
make kind-deploy IMG=local/opendatahub-operator:dev

# 6. Run e2e tests
make e2e-test

# Clean up the kind cluster when done
make kind-delete
```

## Makefile Targets

### Cluster Management

- `make kind-create` - Create a new kind cluster with the default configuration
- `make kind-delete` - Delete the kind cluster
- `make kind-install-olm` - Install OLM (Operator Lifecycle Manager)
- `make kind-install-cert-manager` - Install cert-manager (required for webhook certificates)
- `make kind-setup-webhook-cert` - Create self-signed certificate for operator webhooks
- `make kind-deploy` - Deploy the ODH operator to the kind cluster
- `make kind-load-image` - Load the locally built operator image into the kind cluster
- `make kind-setup` - **Complete setup: create cluster, install OLM, install cert-manager, build image, and deploy operator (one command does it all!)**
- `make kind-status` - Show status of the kind cluster and operator deployment
- `make kind-logs` - Show logs from the operator deployment (follows logs in real-time)
- `make kind-restart` - Restart the operator deployment

### Testing

- `make e2e-test` - Run e2e tests against the cluster (requires cluster setup and operator deployment)

## Configuration

### Kind Cluster Configuration

The kind cluster is created with the following default settings:

- Cluster name: `odh-dev`
- Kubernetes version: Latest stable (configurable via `KIND_K8S_VERSION`)
- Control plane nodes: 1
- Worker nodes: 0 (single-node cluster for faster startup)
- Mock OpenShift version: `4.17.0` (configurable in `config/kind/manager_mock_version_patch.yaml`)

You can customize the cluster configuration by setting environment variables:

```bash
# Use a different cluster name
export KIND_CLUSTER_NAME=my-cluster
make kind-create

# Use a specific Kubernetes version
export KIND_K8S_VERSION=v1.28.0
make kind-create
```

#### Mock OpenShift Version

Since kind runs vanilla Kubernetes (not OpenShift), the operator uses a `MOCK_CLUSTER_VERSION` environment variable to simulate an OpenShift cluster version. This is automatically set to `4.17.0` when deploying to kind.

To use a different version, edit `config/kind/manager_mock_version_patch.yaml` and change the `MOCK_CLUSTER_VERSION` value before deploying.

### E2E Test Configuration

E2e tests can be configured using environment variables. See the [main README](README.md#configuring-e2e-tests) for full details. Common configurations:

```bash
# Run only specific component tests
make e2e-test E2E_TEST_COMPONENT=dashboard

# Skip operator controller tests (useful for debugging)
make e2e-test E2E_TEST_OPERATOR_CONTROLLER=false E2E_TEST_WEBHOOK=false

# Never delete resources after tests (for troubleshooting)
make e2e-test E2E_TEST_DELETION_POLICY=never
```

## Development Workflow

### 1. Build and Deploy Operator

```bash
# Build the operator image
make image-build IMG=local/opendatahub-operator:dev

# Create kind cluster
make kind-create

# Load the image into kind
make kind-load-image IMG=local/opendatahub-operator:dev

# Deploy the operator
make kind-deploy IMG=local/opendatahub-operator:dev
```

### 2. Run E2E Tests

```bash
# Run all e2e tests
make e2e-test

# Run tests for a specific component
make e2e-test E2E_TEST_COMPONENT=dashboard,workbenches
```

### 3. Iterate on Changes

```bash
# After making code changes, rebuild and redeploy
make image-build IMG=local/opendatahub-operator:dev
make kind-load-image IMG=local/opendatahub-operator:dev

# Restart the operator to pick up the new image
make kind-restart

# Or manually delete and redeploy
kubectl delete deployment opendatahub-operator-controller-manager -n opendatahub-operator-system
make kind-deploy IMG=local/opendatahub-operator:dev

# Run tests again
make e2e-test
```

### 4. Clean Up

```bash
# Delete the kind cluster when done
make kind-delete
```

## Troubleshooting

### Quick Debugging Commands

```bash
# Check cluster and operator status
make kind-status

# View operator logs in real-time
make kind-logs

# Check all pods in the operator namespace
kubectl get pods -n opendatahub-operator-system

# Describe a failing pod
kubectl describe pod <pod-name> -n opendatahub-operator-system

# Get events in the operator namespace
kubectl get events -n opendatahub-operator-system --sort-by='.lastTimestamp'
```

### Cluster Creation Fails

If `make kind-create` fails, check:

- Docker is running: `docker ps`
- No existing cluster with the same name: `kind get clusters`
- Sufficient system resources (RAM, disk space)

Delete any existing cluster and try again:
```bash
make kind-delete
make kind-create
```

### Image Not Found in Kind

If the operator fails to start with `ImagePullBackOff`:

1. Ensure the image was built: `docker images | grep opendatahub-operator`
2. Load the image into kind: `make kind-load-image IMG=local/opendatahub-operator:dev`
3. Verify the image is in kind: `docker exec -it odh-dev-control-plane crictl images | grep opendatahub`
4. Check the deployment image reference:
   ```bash
   kubectl get deployment opendatahub-operator-controller-manager -n opendatahub-operator-system -o jsonpath='{.spec.template.spec.containers[0].image}'
   ```

### E2E Tests Timeout

If e2e tests timeout:

1. Increase test timeout: `make e2e-test E2E_TEST_FLAGS="-timeout 60m"`
2. Check operator logs: `make kind-logs`
3. Check cluster resources: `kubectl top nodes` (requires metrics-server)
4. Check for failed pods: `kubectl get pods -A | grep -v Running`

### CRD Installation Issues

If CRDs fail to install:

```bash
# Manually install CRDs
make install

# Verify CRDs are installed
kubectl get crds | grep opendatahub

# Check CRD details
kubectl describe crd datascienceclusters.datasciencecluster.opendatahub.io
```

### Operator Not Starting

If the operator deployment fails to become ready:

```bash
# Check deployment status
kubectl get deployment -n opendatahub-operator-system

# Check pod status and events
kubectl describe pod -l control-plane=controller-manager -n opendatahub-operator-system

# View recent logs
make kind-logs

# Check for webhook certificate issues
kubectl get secret -n opendatahub-operator-system | grep webhook-cert

# If certificate secret is missing, ensure cert-manager is installed
kubectl get pods -n cert-manager

# Reinstall cert-manager if needed
make kind-install-cert-manager
```

**Common Issue**: Pods stuck in `ContainerCreating` with error `secret "opendatahub-operator-controller-webhook-cert" not found`

**Solution**: The operator requires cert-manager to generate webhook certificates. Install it with:
```bash
make kind-install-cert-manager
```

Then delete the existing pods to trigger recreation:
```bash
kubectl delete pods -n opendatahub-operator-system -l control-plane=controller-manager
```

## Advanced Configuration

### Multi-Node Cluster

To create a multi-node cluster, create a custom kind configuration file:

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
```

Then create the cluster with:
```bash
kind create cluster --name odh-dev --config kind-config.yaml
```

### Port Forwarding

To access services running in the kind cluster from your host:

```bash
# Forward dashboard service
kubectl port-forward -n opendatahub svc/odh-dashboard 8080:8080
```

### Using Local Manifests

To test with local component manifests:

```bash
# Get manifests locally
make get-manifests

# Build operator with local manifests
make image-build USE_LOCAL=true IMG=local/opendatahub-operator:dev

# Load and deploy
make kind-load-image IMG=local/opendatahub-operator:dev
make kind-deploy IMG=local/opendatahub-operator:dev
```

## Resources

- [kind Documentation](https://kind.sigs.k8s.io/)
- [ODH Operator README](README.md)
- [E2E Testing Guide](README.md#run-e2e-tests)
- [Component Integration](docs/COMPONENT_INTEGRATION.md)
