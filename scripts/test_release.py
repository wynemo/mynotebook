import base64
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ReleaseTests(unittest.TestCase):
    def test_tag_and_key_must_match(self):
        validate = load("check-release").validate
        public = bytes(range(32))
        secret = base64.b64encode(bytes(32) + public).decode()
        config = {"version": "1.2.3", "updates": {"publicKey": base64.b64encode(public).decode()}}
        self.assertEqual(validate(config, "push", "v1.2.3", secret), "1.2.3")
        with self.assertRaises(ValueError):
            validate(config, "push", "v1.2.4", secret)
        with self.assertRaises(ValueError):
            validate(config, "workflow_dispatch", "main", "")
        with self.assertRaises(ValueError):
            validate(config, "workflow_dispatch", "main", base64.b64encode(bytes(64)).decode())

    def test_linux_packages_require_manifest_and_both_installers(self):
        collect = load("collect-release").collect
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            source = base / "linux-amd64"
            source.mkdir()
            out = base / "out"
            names = ["mynotebook-1.2.3-linux-amd64.tar.gz", "mynotebook_1.2.3_amd64.deb", "mynotebook-1.2.3-1.x86_64.rpm"]
            for name in names:
                (source / name).write_bytes(b"package")
            with self.assertRaises(ValueError):
                collect("linux/amd64", "1.2.3", base, out)
            (source / "update-linux-amd64.json").write_text(json.dumps({"version": "1.2.3"}))
            self.assertEqual(collect("linux/amd64", "1.2.3", base, out), "linux-amd64")
            self.assertEqual(len(list(out.iterdir())), 4)
            (source / names[1]).unlink()
            with self.assertRaises(ValueError):
                collect("linux/amd64", "1.2.3", base, out)


if __name__ == "__main__":
    unittest.main()
