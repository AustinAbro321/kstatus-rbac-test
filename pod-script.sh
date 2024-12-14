cd tmp
tar -xvzf /tmp/go.tar.gz 
cd kstatus
export PATH=/tmp/go/bin:$PATH
export GOPATH=/tmp/go/bin
export GOCACHE=/tmp/.cache/go-build
go mod tidy
go run main.go