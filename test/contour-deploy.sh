helm repo add bitnami https://charts.bitnami.com/bitnami
helm install contour bitnami/contour --namespace projectcontour --create-namespace