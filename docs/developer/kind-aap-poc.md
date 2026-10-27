# Local kind + AAP PoC

This documents the current Flightctl-in-kind setup using the AAP demo installation as its authentication provider. The Helm values snapshot is in [`deploy/helm/flightctl/values.kind-aap.yaml`](../../deploy/helm/flightctl/values.kind-aap.yaml).

## AAP OAuth application

Use a public OAuth application with the authorization-code grant. Keep the client ID in the values file synchronized with that application; a client secret is not used. The AAP application's space-delimited redirect URI list needs the CLI callbacks and the local PoC UI callback:

```text
http://localhost:8080/callback http://127.0.0.1:8080/callback http://localhost:9000/callback
```

The Flightctl Helm UI is disabled in this setup. The UI runs locally on port 9000, with its proxy on port 3001. Its proxy should use `BASE_UI_URL=http://localhost:9000` for the current temporary UI login change.

## Recreate from a clean kind cluster

Run these from the repository root when the `kind` cluster does not already exist. The deployment script loads local images and bootstraps the development chart with Kubernetes auth.

```bash
kind create cluster --name kind --config deploy/kind.yaml
make build-containers
test/scripts/deploy_with_helm.sh
```

## Apply AAP settings

The values file is an overlay for the existing local development Helm release. `--reuse-values` retains its local image names, NodePorts, and development settings. Apply it after the bootstrap above, or to the existing `flightctl` release:

```bash
helm dependency build ./deploy/helm/flightctl
helm upgrade flightctl ./deploy/helm/flightctl \
  --namespace flightctl-external \
  --reuse-values \
  --values ./deploy/helm/flightctl/values.kind-aap.yaml \
  --kube-context kind-kind
```

The current host-specific base domain in the values file is `192.168.1.174.nip.io`; update it if the host address changes. The AAP URL and TLS verification setting are specific to the current local AAP demo. Replace them with the gateway URL and CA configuration for another environment.
