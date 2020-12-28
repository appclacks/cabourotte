#!/bin/bash

version=$1

docker build -t mcorbin/cabourotte:${version} .
docker push mcorbin/cabourotte:${version}
