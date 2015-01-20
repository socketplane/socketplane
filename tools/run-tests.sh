#!/bin/sh

if [ -z "$coveralls_token" ]; then
	make test
else
	make test-all
fi

