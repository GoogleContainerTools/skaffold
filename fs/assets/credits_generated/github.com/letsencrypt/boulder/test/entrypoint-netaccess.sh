#!/bin/bash
# For the boulder container, we want to run entrypoint.sh and start.py by
# default (when no command is passed on a "docker compose run" command line).
# However, we want the netaccess container to run nothing by default.
# Otherwise it would race with boulder container's entrypoint.sh to run
# migrations, and one or the other would fail randomly. Also, it would compete
# with the boulder container for ports. This is a variant of entrypoint.sh that
# exits if it is not given an argument.
if [[ "$@" = "" ]]; then
  echo "Not needed as part of 'docker compose up'. Exiting normally."
  exit 0
fi
"$@"
