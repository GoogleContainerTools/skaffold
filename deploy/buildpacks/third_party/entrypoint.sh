#!/bin/sh
#
# Originally copied from the nodejs/docker-node project's docker-entrypoint.sh.
# The nodejs/docker-node project is made available under the MIT License.
# nodejs/docker-node project: <https://github.com/nodejs/docker-node/tree/aaa6fff33bc11ca04d8e3429c3e08292ca7adfe7>
# docker-entrypoint.sh: <https://github.com/nodejs/docker-node/blob/aaa6fff33bc11ca04d8e3429c3e08292ca7adfe7/docker-entrypoint.sh>
# project LICENSE: <https://github.com/nodejs/docker-node/blob/aaa6fff33bc11ca04d8e3429c3e08292ca7adfe7/LICENSE>
#
# The MIT License (MIT)
# 
# Copyright (c) 2015 Joyent, Inc.
# Copyright (c) 2015 Node.js contributors
# Copyright 2021 The Skaffold Authors
# 
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
# 
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

set -e

if [ "${1#-}" != "${1}" ] || [ -z "$(command -v "${1}")" ]; then
  set -- pack "$@"
fi

exec "$@"

