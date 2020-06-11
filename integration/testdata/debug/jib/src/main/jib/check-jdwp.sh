#!/bin/sh
# Check that JDWP connection exists using `jdb`.  `jdb` never returns
# a non-zero exit code even if it fails to connect, but it does output
# failure messages to stderr.

if [ $# -eq 0 ]; then
  echo "use: $0 <jdwp-port>"
  exit 2
fi

# Attempt to attach and look for jdb attach failure; failure message is output to stdout. 
if jdb -attach $1 < /dev/null 2>&1 | grep 'Unable to attach to target VM'; then
  # failure message found so failed to connect
  exit 1
fi

echo "connected to jdwp on port $1"
exit 0
