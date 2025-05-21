#!/bin/bash

program_version="1.0.0"
compiler_version=$(go version)
build_time=$(date)
author=$(whoami)
go build -ldflags "-X 'main.ProgramVersion=$program_version' -X 'main.CompileVersion=$compiler_version' -X 'main.BuildTime=$build_time' -X 'main.Author=$author'"
