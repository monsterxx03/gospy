all:
	go build  -ldflags "-X main.gitVer=`git describe --tags` -X main.buildAt=`date -u +'%Y-%m-%dT%T%z'`" -o bin/gospy .

test:
	go test

data:
	cd testdata && bash update.sh

