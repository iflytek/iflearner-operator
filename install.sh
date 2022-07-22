echo "create crd"
bin/kustomize build config/crd | kubectl apply -f -

echo "create controller"
# bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && ../../bin/kustomize edit set image controller=iflearner-operator:0.1.0
cd ../.. && bin/kustomize build config/default | kubectl apply -f -

echo "create cert configmap"
kubectl create configmap server-iflearner-crt --from-file=ingress-nginx/server-iflearner-secret.crt
