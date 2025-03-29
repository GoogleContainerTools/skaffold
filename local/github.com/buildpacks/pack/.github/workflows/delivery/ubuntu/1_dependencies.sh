function dependencies() {
    : "$GO_DEP_PACKAGE_NAME"

    echo "> Installing dev tools..."
    apt-get update
    apt-get install gnupg debhelper dput dh-make devscripts lintian software-properties-common -y

    echo "> Installing git..."
    apt-get install git -y

    echo "> Installing go..."
    add-apt-repository ppa:longsleep/golang-backports -y
    apt-get update
    apt-get install $GO_DEP_PACKAGE_NAME -y
}
