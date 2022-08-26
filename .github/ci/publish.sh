#!/bin/sh

PWD_DIR=$(pwd)

go install github.com/ability-sh/ability@v1.0.0

echo $GOPATH

ENV_OS=`$GOPATH/bin/ability env os`
ENV_ARCH=`$GOPATH/bin/ability env arch`

mkdir dist
mkdir dist/cloud
mkdir dist/cloud/bin

mv abi-app-store dist/cloud/bin/$ENV_OS-$ENV_ARCH

$GOPATH/bin/ability app publish 
