#!/bin/bash

set -e

IMAGE_NAME="dmp"

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do 
    SOURCE="$(readlink "$SOURCE")";
done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"
echo $DIR

echo "==> Getting dependencies..."
go get ./...

echo "==> Removing old directory..."
rm -f bin/*
rm -f pkg/*
mkdir -p bin/

echo "==> Building..."
go build -o bin/dmp -pkgdir main

echo "==> Create docker dmp..."
docker build -t ${IMAGE_NAME} .
