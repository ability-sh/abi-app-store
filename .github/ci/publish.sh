#!/bin/sh

PWD_DIR=$(pwd)

echo $PWD_DIR
echo $ABI_TOKEN

go install github.com/ability-sh/ability@v1.0.0

ENV_OS=`ability env os`
ENV_ARCH=`ability env arch`

mkdir dist
mkdir dist/cloud
mkdir dist/cloud/bin
mkdir dist/cloud/bin/$ENV_OS
mkdir dist/cloud/bin/$ENV_OS/$ENV_ARCH

mv abi-app-store dist/cloud/bin/$ENV_OS-$ENV_ARCH

ability app publish 
