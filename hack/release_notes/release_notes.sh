# you can pass your github token with --token here if you run out of requests
go run hack/release_notes/listpullreqs.go

echo "Huge thank you for this release towards our contributors: "
git log "$(git describe  --abbrev=0)".. --format="%aN" --reverse | sort | uniq | awk '{printf "- %s\n", $0 }'
