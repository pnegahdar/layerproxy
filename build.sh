#!/usr/bin/env bash
goxc -os='linux darwin' -arch="amd64" -d=bin/ clean-destination xc archive-tar-gz
