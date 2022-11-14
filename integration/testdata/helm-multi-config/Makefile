build:
	cd skaffold; \
	DOCKER_HOST="unix:///Users/$$(whoami)/.docker/run/docker.sock" \
	skaffold build \
		--verbosity info \
		--default-repo docker.io/bringes

dev:
	cd skaffold; \
	DOCKER_HOST="unix:///Users/$$(whoami)/.docker/run/docker.sock" \
	skaffold dev \
		--verbosity info \
		--default-repo docker.io/bringes \
		--kubeconfig ../kubeconfig.yml

render:
	cd skaffold; \
	DOCKER_HOST="unix:///Users/$$(whoami)/.docker/run/docker.sock" \
	skaffold render \
		--verbosity info \
		--default-repo docker.io/bringes \
        -vdebug
