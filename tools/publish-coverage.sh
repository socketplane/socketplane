#!/bin/sh

[ -z $coveralls_token ] || goveralls -service drone.io -coverprofile=coverage.out -repotoken $coveralls_token 
