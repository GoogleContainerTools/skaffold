#!/usr/bin/env bash
set -e
set -u

# ensure variable is set
: "$PACK_VERSION"
: "$PACKAGE_NAME"
: "$AUR_KEY"
: "$GITHUB_WORKSPACE"

PACKAGE_DIR="${GITHUB_WORKSPACE}/${PACKAGE_NAME}"
PACKAGE_AUR_DIR="${GITHUB_WORKSPACE}/${PACKAGE_NAME}-aur"

# setup non-root user
useradd -m archie

# add non-root user to sudoers
pacman -Sy --noconfirm sudo
echo 'archie ALL=(ALL:ALL) NOPASSWD:ALL' >> /etc/sudoers

echo '> Install dependencies'
pacman -Sy --noconfirm git openssh base-devel libffi

echo '> Configuring ssh...'
SSH_HOME="/root/.ssh"
mkdir -p "${SSH_HOME}"
chmod 700 "${SSH_HOME}"

echo '> Starting ssh-agent...'
eval $(ssh-agent)

echo '> Add Github to known_hosts...'
ssh-keyscan -H aur.archlinux.org >> "${SSH_HOME}/known_hosts"
chmod 644 "${SSH_HOME}/known_hosts"

echo '> Adding AUR_KEY...'
ssh-add - <<< "$AUR_KEY"

echo '> Cloning aur...'
git clone "ssh://aur@aur.archlinux.org/${PACKAGE_NAME}.git" "${PACKAGE_AUR_DIR}"
chown -R archie "${PACKAGE_AUR_DIR}"
pushd "${PACKAGE_AUR_DIR}" > /dev/null
  echo '> Declare directory ${PACKAGE_AUR_DIR} as safe'
  git config --global --add safe.directory "${PACKAGE_AUR_DIR}"

  echo '> Checking out master...'
  git checkout master

  echo '> Applying changes...'
  rm -rf ./*
  cp -R "${PACKAGE_DIR}"/* ./
  
  su archie -c "makepkg --printsrcinfo" > .SRCINFO  
  
  echo '> Committing changes...'
  git config --global user.name "github-bot"
  git config --global user.email "action@github.com"
  git diff --color | cat
  git add .
  git commit -m "Version ${PACK_VERSION}"
  git push -f

popd > /dev/null
