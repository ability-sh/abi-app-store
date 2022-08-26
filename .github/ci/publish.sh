#!/bin/sh

PWD_DIR=$(pwd)

go install github.com/ability-sh/ability@v1.0.0

echo $GOPATH

ABILITY=/root/go/bin/ability

chmod +x $ABILITY

ENV_OS=`$ABILITY env os`
ENV_ARCH=`$ABILITY env arch`

mkdir dist
mkdir dist/cloud
mkdir dist/cloud/bin

mv abi-app-store dist/cloud/bin/$ENV_OS-$ENV_ARCH

$ABILITY app publish 
