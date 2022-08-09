echo "delete controller"
bin/kustomize build config/default | kubectl delete --ignore-not-found=true -f -

echo "delete crd"
bin/kustomize build config/crd | kubectl delete --ignore-not-found=true -f -

echo "delete server cert configmap"
kubectl delete configmap server-iflearner-crt

echo "delete party cert configmap"
kubectl delete configmap party-iflearner-crt
