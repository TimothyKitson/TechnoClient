import argparse
import hashlib
import json
import sys
import urllib.parse
import urllib.request
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PACK_FILE = ROOT / "pack.json"
LOCK_FILE = ROOT / "mods.lock.json"
API = "https://api.modrinth.com/v2"
USER_AGENT = "TimothyKitson/TechnoClient (void-client pack builder)"


def fetch(url):
    request = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(request, timeout=60) as response:
        return response.read()


def fetch_json(url):
    return json.loads(fetch(url))


def load_pack():
    return json.loads(PACK_FILE.read_text())


def resolve(mod, minecraft):
    query = urllib.parse.urlencode({
        "game_versions": json.dumps([minecraft]),
        "loaders": json.dumps(["fabric"]),
    })
    versions = fetch_json(f"{API}/project/{mod['project']}/version?{query}")
    allowed = {"release", "beta"} if mod.get("allow_beta") else {"release"}
    candidates = [v for v in versions if v["version_type"] in allowed]
    if not candidates:
        raise SystemExit(f"{mod['project']} has no {'/'.join(sorted(allowed))} build for Minecraft {minecraft}")
    version = candidates[0]
    file = next((f for f in version["files"] if f["primary"]), version["files"][0])
    required = [d["project_id"] for d in version["dependencies"] if d["dependency_type"] == "required" and d.get("project_id")]
    return {
        "project": mod["project"],
        "project_id": version["project_id"],
        "version": version["version_number"],
        "version_id": version["id"],
        "path": f"mods/{file['filename']}",
        "url": file["url"],
        "sha1": file["hashes"]["sha1"],
        "sha512": file["hashes"]["sha512"],
        "size": file["size"],
        "requires": required,
    }


def lock():
    pack = load_pack()
    entries = [resolve(mod, pack["minecraft"]) for mod in pack["mods"]]
    present = {entry["project_id"] for entry in entries}
    missing = sorted({dep for entry in entries for dep in entry["requires"]} - present)
    if missing:
        names = [fetch_json(f"{API}/project/{pid}")["slug"] for pid in missing]
        raise SystemExit("Missing required dependencies, add them to pack.json: " + ", ".join(names))
    LOCK_FILE.write_text(json.dumps({"minecraft": pack["minecraft"], "mods": entries}, indent="\t") + "\n")
    for entry in entries:
        print(f"{entry['project']:<20} {entry['version']}")


def load_lock():
    pack = load_pack()
    locked = json.loads(LOCK_FILE.read_text())
    if locked["minecraft"] != pack["minecraft"]:
        raise SystemExit("mods.lock.json is for a different Minecraft version, run: python3 pack/build.py lock")
    return pack, locked["mods"]


def verify(data, entry):
    if hashlib.sha512(data).hexdigest() != entry["sha512"]:
        raise SystemExit(f"Hash mismatch for {entry['path']}")


def download(dest):
    _, mods = load_lock()
    dest = Path(dest)
    dest.mkdir(parents=True, exist_ok=True)
    for entry in mods:
        target = dest / Path(entry["path"]).name
        if target.exists() and hashlib.sha512(target.read_bytes()).hexdigest() == entry["sha512"]:
            continue
        data = fetch(entry["url"])
        verify(data, entry)
        target.write_bytes(data)
        print(f"downloaded {target.name}")


def build(jar, version, out_dir):
    pack, mods = load_lock()
    jar = Path(jar)
    index = {
        "formatVersion": 1,
        "game": "minecraft",
        "versionId": version,
        "name": pack["name"],
        "summary": pack["summary"],
        "files": [
            {
                "path": entry["path"],
                "hashes": {"sha1": entry["sha1"], "sha512": entry["sha512"]},
                "env": {"client": "required", "server": "unsupported"},
                "downloads": [entry["url"]],
                "fileSize": entry["size"],
            }
            for entry in mods
        ],
        "dependencies": {
            "minecraft": pack["minecraft"],
            "fabric-loader": pack["fabric_loader"],
        },
    }
    out_dir = Path(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    output = out_dir / f"VoidClient-{version}.mrpack"
    with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("modrinth.index.json", json.dumps(index, indent="\t"))
        archive.write(jar, f"overrides/mods/{jar.name}")
    print(output)


def main():
    parser = argparse.ArgumentParser(description="Build the Void Client modpack")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("lock", help="pick the newest compatible version of every mod")
    download_parser = commands.add_parser("download", help="download the locked mods into a folder")
    download_parser.add_argument("dest")
    build_parser = commands.add_parser("build", help="write the .mrpack")
    build_parser.add_argument("--jar", required=True)
    build_parser.add_argument("--version", required=True)
    build_parser.add_argument("--out", default="dist")
    args = parser.parse_args()
    if args.command == "lock":
        lock()
    elif args.command == "download":
        download(args.dest)
    else:
        build(args.jar, args.version, args.out)


if __name__ == "__main__":
    sys.exit(main())
