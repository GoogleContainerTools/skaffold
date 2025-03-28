#!/usr/bin/env bash
set -e

: ${GITHUB_TOKEN?"Need to set GITHUB_TOKEN env var."}

usage() {
  echo "Usage: "
  echo "  $0 <owner>/<repo> <version>"
  echo "    <owner>/<repo> github repository"
  echo "    <version>  version of release to generate"
  exit 1; 
}

mustHaveExec() {
  local bin=$1
  local e=$(command -v "$bin")
  if [[ -z "$e" ]]; then
    echo "Need '$bin' to be available"
    exit 1
  fi
}

GITHUB_REPO="${1}"
if [[ -z "${GITHUB_REPO}" ]]; then
  echo "Must specify GitHub repository"
  echo
  usage
  exit 1
fi

VERSION="${2}"
if [[ -z "${VERSION}" ]]; then
  echo "Must specify a version"
  echo
  usage
  exit 1
fi

mustHaveExec curl
mustHaveExec jq

release=$(curl -sSL -H "Authorization: token ${GITHUB_TOKEN}" "https://api.github.com/repos/${GITHUB_REPO}/releases/tags/v${VERSION}")
repository=$(curl -sSL -H "Authorization: token ${GITHUB_TOKEN}" "https://api.github.com/repos/${GITHUB_REPO}")

tmpEventFile=$(mktemp)

cat <<EOF > "$tmpEventFile"
{
  "action": "published",
  "release": ${release},
  "repository": ${repository},
  "sender": {
    "login": "Codertocat",
    "id": 21031067,
    "node_id": "MDQ6VXNlcjIxMDMxMDY3",
    "avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/Codertocat",
    "html_url": "https://github.com/Codertocat",
    "followers_url": "https://api.github.com/users/Codertocat/followers",
    "following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
    "gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
    "organizations_url": "https://api.github.com/users/Codertocat/orgs",
    "repos_url": "https://api.github.com/users/Codertocat/repos",
    "events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
    "received_events_url": "https://api.github.com/users/Codertocat/received_events",
    "type": "User",
    "site_admin": false
  }
}
EOF

echo "$tmpEventFile"