#!/usr/bin/env python3
import sys
import os
stdin_data = sys.stdin.read()
count = os.getenv("REPLICAS")

print(stdin_data.replace("replicas: 1", "replicas: {}".format(count)))