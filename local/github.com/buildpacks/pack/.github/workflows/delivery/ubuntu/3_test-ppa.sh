function test_ppa {
    : "$GITHUB_WORKSPACE"

    echo "> Creating a test directory..."
    testdir="$(mktemp -d)"

    echo "> Source Dir: '$GITHUB_WORKSPACE'"
    echo "> Test Dir: '$testdir'"
    cp -R $GITHUB_WORKSPACE/* $testdir

    pushd $testdir
        echo "> Building a debian binary package..."
        debuild -b -us -uc

        echo "> Installing binary package..."
        dpkg -i ../*.deb

        echo "> Contents installed by the build debain package:"
        dpkg -L pack-cli
    popd
}
