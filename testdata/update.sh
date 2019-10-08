#!/usr/bin/env bash
set -euo pipefail
GOVERSIONS_DIR=${HOME}/Downloads/goversions
GOVERSIONS=(
https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz
https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
https://dl.google.com/go/go1.11.13.linux-amd64.tar.gz
https://dl.google.com/go/go1.10.8.linux-amd64.tar.gz
https://dl.google.com/go/go1.10.linux-amd64.tar.gz
)
orig_dir=$(pwd)

function sync {
    mkdir -p $GOVERSIONS_DIR

    cd $GOVERSIONS_DIR
    for link in ${GOVERSIONS[@]} ; do
        filename=$(basename $link)
        versiondir=$(basename $link .tar.gz)
        if [[ ! -f ${GOVERSIONS_DIR}/${filename} ]]; then
            wget $link -O ${GOVERSIONS_DIR}/${filename}
        fi
        if [[ ! -d ${GOVERSIONS_DIR}/${versiondir} ]]; then
            tar xvf $filename  && mv go $versiondir
        fi
        export GOROOT=${GOVERSIONS_DIR}/${versiondir}
        export PATH=$GOROOT/bin:$PATH
        version=${versiondir#"go"}
        version=${version%".linux-amd64"}
        version=${version//./_}
        $(cd $orig_dir && go build -o test_${version} test.go)
    done
}

sync

