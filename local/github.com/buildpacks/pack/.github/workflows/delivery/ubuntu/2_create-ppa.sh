function create_ppa() {
  # verify the following are set.
  : "$GPG_PUBLIC_KEY"
  : "$GPG_PRIVATE_KEY"
  : "$PACKAGE_VERSION"
  : "$PACKAGE_NAME"
  : "$MAINTAINER_NAME"
  : "$MAINTAINER_EMAIL"
  : "$SCRIPT_DIR"

  echo "> Importing GPG keys..."
  gpg --import <(echo "$GPG_PUBLIC_KEY")
  gpg --allow-secret-key-import --import <(echo "$GPG_PRIVATE_KEY")

  # Dependencies fail to be pulled in during the Launchpad build process.
  echo "> Vendoring dependencies..."
  go mod vendor

  echo "> Creating package: ${PACKAGE_NAME}_${PACKAGE_VERSION}"
  echo "> Generating skeleton of a debian package..."
  export DEBEMAIL=$MAINTAINER_EMAIL
  export DEBFULLNAME=$MAINTAINER_NAME
  dh_make -p "${PACKAGE_NAME}_${PACKAGE_VERSION}" --single --native --copyright apache --email "${MAINTAINER_EMAIL}" -y

  echo "> Copying templated configuration files..."
  cp "$SCRIPT_DIR/debian/"* debian/

  echo "======="
  echo "compat"
  echo "======="
  cat debian/compat
  echo
  echo "======="
  echo "changelog"
  echo "======="
  cat debian/changelog
  echo
  echo "======="
  echo "control"
  echo "======="
  cat debian/control
  echo
  echo "======="
  echo "rules"
  echo "======="
  cat debian/rules
  echo
  echo "======="
  echo "copyright"
  echo "======="
  cat debian/copyright
  echo

  echo "> Removing empty default files created by dh_make..."
  rm -f debian/*.ex
  rm -f debian/*.EX
  rm -f debian/README.*

  # Ubuntu ONLY accepts source packages.
  echo "> Build a source based debian package..."
  debuild -S

  # debuild places everything in parent directory
  echo "> Files created in: ${PWD}/.."
  ls -al ${PWD}/..
}
