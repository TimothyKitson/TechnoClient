#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION=$(grep '^version=' gradle.properties | cut -d= -f2)
MINECRAFT=$(python3 -c 'import json; print(json.load(open("pack/pack.json"))["minecraft"])')
LOADER=$(python3 -c 'import json; print(json.load(open("pack/pack.json"))["fabric_loader"])')

./gradlew build "$@"
python3 pack/build.py build --jar "build/libs/voidclient-$VERSION.jar" --version "$VERSION" --out dist

mkdir -p installer/payload
cp "build/libs/voidclient-$VERSION.jar" installer/payload/voidclient.jar
cp pack/mods.lock.json installer/payload/mods.lock.json
printf '%s' "$VERSION" > installer/payload/version.txt
curl -fsSL "https://meta.fabricmc.net/v2/versions/loader/$MINECRAFT/$LOADER/profile/json" -o installer/payload/fabric-profile.json
cp installer/winres/icon128.png installer/payload/profile-icon.png

(
	cd installer
	go run github.com/tc-hib/go-winres@v0.3.3 make --in winres/winres.json --arch amd64 --product-version "$VERSION" --file-version "$VERSION"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../dist/VoidClient-Installer.exe .
)

cp dist/VoidClient-Installer.exe site/VoidClient-Installer.exe
cp "dist/VoidClient-$VERSION.mrpack" site/VoidClient.mrpack

# Void Launcher (standalone modpack launcher)
(
	cd launcher
	go run github.com/tc-hib/go-winres@v0.3.3 make --in winres/winres.json --arch amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../dist/VoidLauncher.exe .
	rm -f *.syso
)
cp dist/VoidLauncher.exe site/VoidLauncher.exe

echo "site/ is ready to upload to Netlify"
