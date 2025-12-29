# Launch Readiness Checklist

## ✅ Launch Blockers - FIXED

### 1. Helm Chart Default Install ✅
- **Fixed**: `webhook.enabled: false` by default
- **Fixed**: `failurePolicy: Ignore` by default (safe for alpha)
- **Result**: `helm install zen-flow charts/zen-flow` works on clean cluster

### 2. Single Canonical Install Story ✅
- **Primary**: Helm installation (recommended)
- **Alternative**: kubectl installation (documented as alternative)
- **Result**: Clear, single path for users

### 3. Helm RBAC Alignment ✅
- **Fixed**: Added `delete` permission for `jobflows` (TTL cleanup, Replace policy)
- **Fixed**: Added `delete` permission for `jobs` (step timeout cleanup)
- **Result**: Helm RBAC matches controller behavior

## ✅ High-Priority OSS Readiness - FIXED

### 4. Webhook Safety Defaults ✅
- **Fixed**: `webhook.enabled: false` by default
- **Fixed**: `failurePolicy: Ignore` by default (configurable)
- **Result**: Non-bricking defaults for alpha launch

### 5. Security Posture Defaults ✅
- **Already present**: `runAsNonRoot: true`, `runAsUser: 65534` in values.yaml
- **Already present**: `fsGroup: 65534`, `seccompProfile: RuntimeDefault`
- **Result**: Aligns with Pod Security Standards

### 6. Complete Quick Start ✅
- **Fixed**: README.md rewritten with Helm-first approach
- **Includes**: Installation, verification, example, troubleshooting, uninstall
- **Result**: Single source of truth for getting started

### 7. Release Artifact Clarity ✅
- **Docker Images**: `docker.io/kubezen/zen-flow-controller`
- **Tags**: `0.0.1-alpha`, `latest`
- **Helm Chart**: Local (`./charts/zen-flow/`) - ready for publishing
- **Chart Version**: `0.0.1-alpha`
- **App Version**: `0.0.1-alpha`
- **Result**: Clear publishing locations documented

## OSS Launch Gate Checklist

### ✅ Test on Clean Cluster

```bash
# 1. Create clean cluster
kind create cluster --name zen-flow-test

# 2. Helm install with defaults
helm install zen-flow ./charts/zen-flow --namespace zen-flow-system --create-namespace

# 3. Verify installation
kubectl get pods -n zen-flow-system
kubectl get crd jobflows.workflow.zen.io

# 4. Apply example
kubectl apply -f examples/simple-linear-flow.yaml

# 5. Watch JobFlow reach Succeeded
kubectl get jobflow simple-linear-flow -w

# 6. Uninstall
helm uninstall zen-flow --namespace zen-flow-system
```

### ✅ Optional: Test Webhooks

```bash
# Enable webhooks with cert-manager
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true \
  --set webhook.certManager.issuer.name=your-issuer \
  --set webhook.failurePolicy=Fail

# Test validation
kubectl apply -f examples/simple-linear-flow.yaml
```

## Repository Configuration ✅

1. **GitHub Pages Helm Repository** ✅
   - Repository URL: `https://kube-zen.github.io/zen-flow/charts`
   - GitHub Actions workflow configured to publish automatically
   - Charts published to `docs/charts/` directory

2. **Helm Chart Publishing** ✅
   - Charts automatically published to GitHub Pages
   - Repository URL: `https://kube-zen.github.io/zen-flow/charts`

3. **Next Steps**
   - Enable GitHub Pages in repository settings (Settings → Pages)
   - Set source to "GitHub Actions"

## Files Changed

- `charts/zen-flow/values.yaml` - Webhook disabled by default, failurePolicy: Ignore
- `charts/zen-flow/templates/rbac.yaml` - Added delete permissions
- `charts/zen-flow/templates/webhook.yaml` - failurePolicy from values
- `charts/zen-flow/Chart.yaml` - Release info comments
- `README.md` - Complete Helm-first Quick Start
- `docs/OPERATOR_GUIDE.md` - Helm-first installation
- `CHANGELOG.md` - Documented all fixes

## Status: ✅ READY FOR LAUNCH

All launch blockers and high-priority items are fixed. The project is ready for:
- GitHub Helm chart repository configuration
- OSS announcement

