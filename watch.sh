#!/bin/bash

# Function to kill the previously running instance, if it exists
kill_app() {
  if [[ ! -z "$app_pid" ]]; then
    echo "Stopping previous instance (PID: $app_pid)..."
    kill $app_pid
    wait $app_pid 2>/dev/null
  fi
}

# Initial build and run
templ generate
make build
make run &
app_pid=$!  # Save the PID of the app

# Watch for changes in Go source files and .templ files
fswatch -o *.go *.templ | while read file; do
  echo "File change detected in: $file"
  echo "Rebuilding..."

  # Kill previous instance
  kill_app

  # Rebuild and run new instance
  templ generate
  make build
  make run &
  app_pid=$!  # Save the new app PID
done
