# quick and dirty script
kubectl apply -f k8s/
sleep 1
# I know haha
kubectl apply -f k8s/
sleep 10
kubectl cp go1.23.4.linux-amd64.tar.gz kubectl-go-test:/tmp/go.tar.gz -n podinfo
kubectl cp . kubectl-go-test:/tmp/kstatus -n podinfo

# kubectl exec -it kubectl-go-test -n podinfo -- /bin/bash
