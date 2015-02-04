#!/bin/bash

echo "mode: count" > socketplane.coverprofile
cat *.cover.out | grep -v mode: | sort -r | awk '{if($1 != last) {print $0;last=$1}}' >> socketplane.coverprofile
