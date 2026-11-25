"""Collect a single platform's installers and signed update files."""
import json
import os
from pathlib import Path
import shutil


def collect(platform, version, base=Path("build"), out=Path("artifacts")):
    target = platform.replace("/", "-")
    source = base / target
    manifest = source / f"update-{target}.json"
    if not manifest.is_file():
        raise ValueError(f"Signed update manifest is missing: {manifest}")
    if json.loads(manifest.read_text())["version"] != version:
        raise ValueError("Update manifest version does not match the release")
    arch = platform.split("/")[1]
    patterns = [f"mynotebook-{version}-{target}.tar.gz"]
    if platform.startswith("darwin/"):
        patterns += [f"mynotebook-{version}-{arch}.dmg"]
    elif platform.startswith("windows/"):
        patterns += [f"mynotebook-setup-{version}-{arch}.exe"]
    elif platform.startswith("linux/"):
        deb_version = version.replace("-", "~", 1)
        patterns += [f"mynotebook_{deb_version}_{arch}.deb", "mynotebook-*.rpm"]
    files = [manifest]
    for pattern in patterns:
        matches = sorted(source.glob(pattern))
        if not matches:
            raise ValueError(f"Required package is missing: {source / pattern}")
        files.extend(matches)
    files.extend(sorted(source.glob("mynotebook-*.delta")))
    out.mkdir(parents=True, exist_ok=True)
    for file in files:
        shutil.copy2(file, out / file.name)
    return target


if __name__ == "__main__":
    target = collect(os.environ["PLATFORM"], os.environ["VERSION"])
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        output.write(f"target={target}\n")
