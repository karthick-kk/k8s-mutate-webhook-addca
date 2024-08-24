## Gitea install
helm install gitea bitnami/gitea --namespace gitea --create-namespace --set service.type=ClusterIP --set rootURL=https://gitea.company
if [ $(command -v mkcert) ]; then
  cd ssl && mkcert gitea.company
else
  echo "mkcert not found, please install it"
  exit 1
fi

kubectl create secret tls gitea-tls --cert=ssl/gitea.company.pem --key=ssl/gitea.company-key.pem -n gitea 

# Below requires contour to be installed
cat <<EOF | kubectl apply -f -
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: gitea-ingress
  namespace: gitea
spec:
  ingressClassName: contour
  virtualhost:
    fqdn: gitea.company
    tls:
      secretName: gitea-tls
  routes:
    - conditions:
        - prefix: /
      services:
        - name: gitea
          port: 80    
EOF