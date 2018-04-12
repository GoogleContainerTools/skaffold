wrappedNode(label: 'linux && x86_64', cleanWorkspace: true) {
  timeout(time: 60, unit: 'MINUTES') {
    stage "Git Checkout"
    checkout scm

    stage "Run end-to-end test suite"
    sh "docker version"
    sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
        IMAGE_TAG=clie2e${BUILD_NUMBER} \
        make -f docker.Makefile test-e2e"
  }
}
