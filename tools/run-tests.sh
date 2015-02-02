#!/bin/sh

if [ -z "$coveralls_token" ]; then
	make test-local
else
	make test-all-local
fi

