all:
	GOOS=linux GOARCH=amd64 go build  -ldflags "-X main.gitVer=`git describe --tags` -X main.buildAt=`date -u +'%Y-%m-%dT%T%z'`" -o bin/gospy .

test_local:
	go build -o testdata/test_bin testdata/test.go &&  go test

test:
	go test -v
