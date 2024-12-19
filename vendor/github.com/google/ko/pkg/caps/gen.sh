#!/usr/bin/env bash

# Copyright 2024 ko Build Authors All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script assigns different capabilities to files and captures
# resulting xattr blobs for testing (generates caps_dd_test.go).
#
# It has to be run on a reasonably recent Linux to ensure that the full
# set of capabilities is supported. Setting capabilities requires
# privileges; the script assumes paswordless sudo is available.

set -o errexit
set -o nounset
set -o pipefail
shopt -s inherit_errexit

# capblob CAP_STRING
# Obtain base64-encoded value of the underlying xattr that implemens
# specified capabilities, setcap syntax.
# Example: capblob cap_chown=eip
capblob() {
  f=$(mktemp)
  sudo -n setcap $1 $f
  getfattr -n security.capability --absolute-names --only-values $f | base64
  rm $f
}

(
  license=$(sed -e '/^$/,$d' caps.go)

  echo "// Generated file, do not edit."
  echo ""
  echo "$license"
  echo ""
  echo "package caps"
  echo "var ddTests = []ddTest{"

  res=$(capblob cap_chown=p)
  echo "{permitted: \"chown\", inheritable: \"\", effective: false, res: \"$res\"},"

  res=$(capblob cap_chown=ep)
  echo "{permitted: \"chown\", inheritable: \"\", effective: true, res: \"$res\"},"

  res=$(capblob cap_chown=i)
  echo "{permitted: \"\", inheritable: \"chown\", effective: false, res: \"$res\"},"

  CAPS="chown dac_override dac_read_search fowner fsetid kill setgid setuid
    setpcap linux_immutable net_bind_service net_broadcast net_admin net_raw ipc_lock ipc_owner
    sys_module sys_rawio sys_chroot sys_ptrace sys_pacct sys_admin sys_boot sys_nice
    sys_resource sys_time sys_tty_config mknod lease audit_write audit_control setfcap
    mac_override mac_admin syslog wake_alarm block_suspend audit_read perfmon bpf
    checkpoint_restore"
  for cap in $CAPS; do
    res=$(capblob cap_$cap=eip)
    echo "{permitted: \"$cap\", inheritable: \"$cap\", effective: true, res: \"$res\"},"
  done

  echo "}"
) > caps_dd_test.go

gofmt -w -s ./caps_dd_test.go
