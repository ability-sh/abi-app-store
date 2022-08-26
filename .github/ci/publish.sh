#!/bin/sh

PWD_DIR=$(pwd)

ABILITY=./.github/ci/ability

ENV_OS=`$ABILITY env os`
ENV_ARCH=`$ABILITY env arch`

mkdir dist
mkdir dist/bin

mv abi-app-store dist/bin/$ENV_OS-$ENV_ARCH

echo "$ABILITY app publish -token $ABI_TOKEN -file ./app.yaml -number $ABI_NUMBER"

$ABILITY app publish -token $ABI_TOKEN -file ./app.yaml -number $ABI_NUMBER
