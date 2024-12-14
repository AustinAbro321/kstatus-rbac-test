# quick and dirty script, not meant to be run because we exec into a pod halfway through
k apply -f k8s/
kubectl cp go1.23.4.linux-amd64.tar.gz kubectl-go-test:/tmp/go.tar.gz -n podinfo
kubectl cp . kubectl-go-test:/tmp/kstatus -n podinfo

kubectl exec -it kubectl-go-test -n podinfo -- /bin/bash
cd tmp
tar -xvzf /tmp/go.tar.gz 
cd kstatus
export PATH=/tmp/go/bin:$PATH
export GOPATH=/tmp/go/bin
export GOCACHE=/tmp/.cache/go-build
go mod tidy
go run main.go