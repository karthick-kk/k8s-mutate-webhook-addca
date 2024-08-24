# Authentik install
helm repo add goauthentik https://charts.goauthentik.io/
helm install authentik -n authentik --create-namespace goauthentik/authentik --set authentik.secret_key=mySecretk3y --set postgresql.enabled=true --set postgresql.auth.password=secret123 --set authentik.postgresql.password=secret123 --set redis.enabled=true
if [ $(command -v mkcert) ]; then
  cd ssl && mkcert authentik.company
else
  echo "mkcert not found, please install it"
  exit 1
fi
kubectl create secret tls authentik-tls --cert=ssl/authentik.company.pem --key=ssl/authentik.company-key.pem -n authentik

# Below requires contour to be installed
cat <<EOF | kubectl apply -f -
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: authentik-ingress
  namespace: authentik
spec:
  ingressClassName: contour
  virtualhost:
    fqdn: authentik.company
    tls:
      secretName: authentik-tls
  routes:
    - conditions:
        - prefix: /
      enableWebsockets: true
      services:
        - name: authentik-server
          port: 80    
EOF