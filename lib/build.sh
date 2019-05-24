#!/bin/sh

CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildmode=c-archive -ldflags '-s -w' -o /Volumes/HDD/sourceRepo/vpndemo_oc/htunclient/amd64/htunclient.a

CC=/usr/local/Cellar/go/1.11.4/libexec/misc/ios/clangwrap.sh CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildmode=c-archive -ldflags '-s -w' -o /Volumes/HDD/sourceRepo/vpndemo_oc/htunclient/arm64/htunclient.a