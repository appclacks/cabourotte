#!/bin/bash

version=$1
tag -a ${version} -m "release ${version}"

docker build -t mcorbin/cabourotte:${version} .
docker push mcorbin/cabourotte:${version}
