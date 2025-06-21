#!/bin/bash

# Run all tests
go test ./...

# Check the exit code of the tests
if [ $? -eq 0 ]; then
  echo "Tests passed!"
else
  echo "Tests failed!"
fi
