#!/bin/bash

# Watch for changes in Go source files
fswatch -o *.go | while read; do
  make build
  make run
done
