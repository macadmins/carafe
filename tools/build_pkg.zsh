#!/bin/zsh

set -e

/bin/rm -rf payload
/bin/rm -rf build
/bin/rm -rf output
/bin/mkdir -p payload/opt/macadmins/bin
/bin/mkdir -p build
/bin/mkdir -p output

XCODE_PATH="/Applications/Xcode_16.4.app"
APP_SIGNING_IDENTITY="Developer ID Application: Mac Admins Open Source (T4SK8ZXCXG)"
INSTALLER_SIGNING_IDENTITY="Developer ID Installer: Mac Admins Open Source (T4SK8ZXCXG)"
XCODE_NOTARY_PATH="$XCODE_PATH/Contents/Developer/usr/bin/notarytool"
XCODE_STAPLER_PATH="$XCODE_PATH/Contents/Developer/usr/bin/stapler"

# read the version from passed argument
VERSION=$1

# ensure xcode is installed
if [ ! -d "$XCODE_PATH" ]; then
  echo "Xcode not found at $XCODE_PATH"
  exit 1
fi

# ensure the notary tool is installed
if [ ! -f "$XCODE_NOTARY_PATH" ]; then
  echo "Notary tool not found at $XCODE_NOTARY_PATH"
  exit 1
fi

# ensure the stapler tool is installed
if [ ! -f "$XCODE_STAPLER_PATH" ]; then
  echo "Stapler tool not found at $XCODE_STAPLER_PATH"
  exit 1
fi

# Ensure Xcode is set to run-time
sudo xcode-select -s "$XCODE_PATH"

echo "Building version ${VERSION}"

# build arm64
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${VERSION}" -o build/carafe-arm64

# build amd64
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o build/carafe-amd64

# combine the two into a universal binary
echo "Combining arm64 and amd64 into a universal binary"
/usr/bin/lipo -create -output build/carafe build/carafe-arm64 build/carafe-amd64

# copy the binary to the payload
echo "Copying the binary to the payload"
/bin/cp build/carafe payload/opt/macadmins/bin/carafe

# sign the binary
echo "Signing the binary"
sudo /usr/bin/codesign --timestamp --force --deep -s "${APP_SIGNING_IDENTITY}" --options=runtime --entitlements ./tools/carafe.entitlement payload/opt/macadmins/bin/carafe

# create the package
echo "Creating the package"
/usr/bin/pkgbuild --root payload --identifier com.github.macadmins.carafe --version ${VERSION} --install-location / --ownership recommended --sign "${INSTALLER_SIGNING_IDENTITY}" output/carafe.pkg

# notarize the package
echo "Notarizing the package with ${XCODE_NOTARY_PATH}"
$XCODE_NOTARY_PATH store-credentials --apple-id "opensource@macadmins.io" --team-id "T4SK8ZXCXG" --password "$2" carafe

# Notarize carafe package
$XCODE_NOTARY_PATH submit "output/carafe.pkg" --keychain-profile "carafe" --wait
$XCODE_STAPLER_PATH staple "output/carafe.pkg"