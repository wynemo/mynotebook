"""Validate release inputs before any runner builds or publishes files."""
import base64
import json
import os
from pathlib import Path
import re


def validate(config, event, tag, key):
    version = config["version"]
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?", version):
        raise ValueError("mygo.json must contain a semantic version")
    if event == "push" and tag != "v" + version:
        raise ValueError(f"Tag {tag} does not match mygo.json version {version}")
    if not key.strip():
        raise ValueError("Repository secret MYGO_UPDATER_PRIVATE_KEY is missing")
    private = base64.b64decode(key.strip(), validate=True)
    public = base64.b64decode(config["updates"]["publicKey"], validate=True)
    # MyGo stores Ed25519 private keys as the 32-byte seed + public key.
    if len(private) != 64 or len(public) != 32 or private[32:] != public:
        raise ValueError("Update signing key does not match mygo.json public key")
    return version


if __name__ == "__main__":
    config = json.loads(Path("mygo.json").read_text())
    version = validate(config, os.environ["EVENT_NAME"], os.environ["TAG"], os.environ.get("MYGO_UPDATER_PRIVATE_KEY", ""))
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        output.write(f"version={version}\nprerelease={'true' if '-' in version else 'false'}\n")
