#!/bin/bash

version=$1

docker build -t appclacks/cabourotte:${version} .
docker push appclacks/cabourotte:${version}
