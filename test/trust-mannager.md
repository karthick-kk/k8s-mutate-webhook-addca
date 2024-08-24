# Install Trust Manager
```bash
helm repo add jetstack https://charts.jetstack.io --force-update
helm upgrade trust-manager jetstack/trust-manager \        
  --install \
  --namespace cert-manager --create-namespace \
  --wait \
  --set app.webhook.tls.helmCert.enabled=true
```
# Add custom root CA secret
```bash
kubectl create secret generic root-ca-secret --from-file=ca.crt=/home/kk/.local/share/mkcert/rootCA.pem -n cert-manager
```
# Create a trust bundle
```bash
kubectl apply -f - <<EOF
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: trust-bundle
spec:
  sources:
  - useDefaultCAs: true
  - secret:
      name: "root-ca-secret"
      key: "ca.crt"
  target:
    configMap:
      key: "trust-bundle.pem"
EOF
```
# Label namespace
```
kubectl label namespace default mutateme=enable
kubectl label namespace default trust=enabled
kubectl get ns -L trust -L mutateme
```


