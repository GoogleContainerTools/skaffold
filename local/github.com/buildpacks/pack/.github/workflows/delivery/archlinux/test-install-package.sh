#!/usr/bin/env bash
set -e
set -u

# ensure variable is set
: "$PACKAGE_NAME"
: "$GITHUB_WORKSPACE"

# setup non-root user
useradd -m archie

# add non-root user to sudoers
pacman -Sy --noconfirm sudo
echo 'archie ALL=(ALL:ALL) NOPASSWD:ALL' >> /etc/sudoers

# setup workspace
WORKSPACE=$(mktemp -d -t "$PACKAGE_NAME-XXXXXXXXXX")
cp -R "$GITHUB_WORKSPACE/$PACKAGE_NAME/"* "$WORKSPACE"
chown -R archie "$WORKSPACE"

# run everything else as non-root user
pushd "$WORKSPACE" > /dev/null
su archie << "EOF"
echo -n '> Debug info:'
ls -al
sha512sum ./*

echo -n '> Installing AUR packaging deps...'
sudo pacman -Sy --noconfirm git base-devel libffi

echo -n '> Installing package...'
makepkg -sri --noconfirm

# print version
echo -n '> Installed pack version: '
pack --version
EOF
popd > /dev/null
