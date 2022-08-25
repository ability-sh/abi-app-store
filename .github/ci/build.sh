#!/bin/sh

PWD_DIR=$(pwd)

echo $PWD_DIR

docker run --rm -v $PWD_DIR:/workdir -w "/workdir" docker.io/nginx/unit:1.27.0-go1.18 /workdir/.github/ci/docker_build.sh
