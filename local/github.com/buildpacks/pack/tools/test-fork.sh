#!/usr/bin/env bash

readonly wfdir=".github/workflows"

# $1 - registry repo name

echo "Parse registry: $1"
firstPart=$(echo "$1" | cut -d/ -f1)
secondPart=$(echo "$1" | cut -d/ -f2)
thirdPart=$(echo "$1" | cut -d/ -f3)

registry=""
username=""
reponame=""
if [[ -z $thirdPart ]]; then # assume Docker Hub
  registry="index.docker.io"
  username=$firstPart
  reponame=$secondPart
else
  registry=$firstPart
  username=$secondPart
  reponame=$thirdPart
fi

echo "Using registry $registry and username $username"
if [[ $reponame != "pack" ]]; then
  echo "Repo name must be 'pack'"
  exit 1
fi

echo "Disabling workflows that should not run on the forked repository"
disable=(
  delivery-archlinux-git.yml
  delivery-archlinux.yml
  delivery-chocolatey.yml
  delivery-homebrew.yml
  delivery-release-dispatch.yml
  delivery-ubuntu.yml
  privileged-pr-process.yml
)
for d in "${disable[@]}"; do
  if [ -e "$wfdir/$d" ]; then
    mv "$wfdir/$d" "$wfdir/$d.disabled"
  fi
done

echo "Removing upstream maintainers from the benchmark alert CC"
sed -i '' "/alert-comment-cc-users:/d" $wfdir/benchmark.yml

echo "Removing the architectures that require self-hosted runner from the build strategies."
sed -i '' "/config: \[.*\]/ s/windows-lcow, //g" $wfdir/build.yml
sed -i '' "/- config: windows-lcow/,+4d" $wfdir/build.yml

echo "Replacing the registry account with owned one (assumes DOCKER_PASSWORD and DOCKER_USERNAME have been added to GitHub secrets, if not using ghcr.io)"
sed -i '' "s/buildpacksio\/pack/$registry\/$username\/$reponame/g" $wfdir/check-latest-release.yml
sed -i '' "/REGISTRY_NAME: 'index.docker.io'/ s/index.docker.io/$registry/g" $wfdir/delivery-docker.yml
sed -i '' "/USER_NAME: 'buildpacksio'/ s/buildpacksio/$username/g" $wfdir/delivery-docker.yml

if [[ $registry != "index.docker.io" ]]; then
  echo "Updating login action to specify the registry"
  sed -i '' "s/username: \${{ secrets.DOCKER_USERNAME }}/registry: $registry\n          username: $username/g" $wfdir/delivery-docker.yml
fi

if [[ $registry == *"ghcr.io"* ]]; then
  echo "Updating login action to use GitHub token for ghcr.io"
  sed -i '' "s/secrets.DOCKER_PASSWORD/secrets.GITHUB_TOKEN/g" $wfdir/delivery-docker.yml

  echo "Adding workflow permissions to push images to ghcr.io"
  LF=$'\n'
  sed -i '' "/runs-on: ubuntu-latest/ a\\
    permissions:\\
      contents: read\\
      packages: write\\
      attestations: write\\
      id-token: write${LF}" $wfdir/delivery-docker.yml
  LF=""
fi
