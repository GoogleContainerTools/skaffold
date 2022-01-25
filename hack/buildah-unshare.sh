#!/bin/sh

buildah unshare $HOME/go/bin/dlv-dap dap --listen=127.0.0.1:2345
