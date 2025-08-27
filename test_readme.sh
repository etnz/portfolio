#!/bin/bash
set -ex
# Build the pcs binary
go build -o /tmp/pcs ./pcs/

# Create a fresh empty folder
rm -rf /tmp/test-readme-run
mkdir -p /tmp/test-readme-run

# Change the current directory to the new folder
cd /tmp/test-readme-run

# Run the commands
/tmp/pcs add-security -s AAPL -id US0378331005.XETR -c EUR
/tmp/pcs add-security -s "My Corporate Fund" -id "MYCORPFUND" -c EUR
/tmp/pcs declare -s AAPL -id US0378331005.XETR -c EUR
/tmp/pcs declare -s "My Corporate Fund" -id "MYCORPFUND" -c EUR
/tmp/pcs deposit -a 10000 -c EUR
/tmp/pcs buy -s AAPL -q 10 -p 150.0
/tmp/pcs update
/tmp/pcs holding
/tmp/pcs summary

# Clean up
cd ..
rm -rf /tmp/test-readme-run
