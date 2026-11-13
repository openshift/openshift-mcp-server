import ctypes
import os
from pathlib import Path
import re
import signal
import socket
import subprocess
import sys
import tempfile
import time
import unittest


class EvalStartup(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repository = Path(__file__).resolve().parent.parent
        cls.temporary = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.root = Path(cls.temporary.name).resolve()
        cls.server = cls.root / "server"
        subprocess.run(["go", "build", "-o", str(cls.server), "./cmd/kubernetes-mcp-server"], cwd=cls.repository, check=True)
        if sys.platform == "linux":
            if ctypes.CDLL(None, use_errno=True).prctl(36, 1, 0, 0, 0) != 0:
                raise RuntimeError("could not enable owned server process reaping")
        cls.kubeconfig = cls.root / "kubeconfig"
        cls.kubeconfig.write_text(
            "apiVersion: v1\nkind: Config\nclusters:\n- name: owned\n"
            "  cluster:\n    server: http://127.0.0.1:1\ncontexts:\n- name: owned\n"
            "  context:\n    cluster: owned\n    user: owned\ncurrent-context: owned\n"
            "users:\n- name: owned\n  user:\n    token: owned-fixture-token\n"
        )

    def configuration(self):
        return 'port = "0"\nkubeconfig = "%s"\ncluster_provider_strategy = "kubeconfig"\ntoolsets = ["core", "config"]\n' % self.kubeconfig

    def stop_server(self, pid):
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            return
        deadline = time.monotonic() + 5
        while True:
            try:
                child, _ = os.waitpid(pid, os.WNOHANG)
            except ChildProcessError:
                return
            if child:
                return
            if time.monotonic() >= deadline:
                os.kill(pid, signal.SIGKILL)
                os.waitpid(pid, 0)
                return
            time.sleep(0.01)

    def test_actual_make_copies_dropins_and_starts_server(self):
        working = self.root / "make-consumer"
        working.mkdir()
        (working / "server").symlink_to(self.server)
        with socket.socket() as probe:
            probe.bind(("127.0.0.1", 0))
            port = probe.getsockname()[1]
        source = self.repository / "dev/config/mcp-configs"
        generated = working / "generated"
        environment = dict(os.environ, NO_PROXY="127.0.0.1,localhost,::1", no_proxy="127.0.0.1,localhost,::1")
        with tempfile.TemporaryFile() as output:
            try:
                result = subprocess.run([
                    "make", "-f", str(self.repository / "build/evals.mk"), "-o", "build", "run-server",
                    "BINARY_NAME=server", "MCP_CONFIG_DIR=" + str(source), "MCP_EVAL_CONFIG_DIR=" + str(generated),
                    "MCP_EVAL_KUBECONFIG=" + str(self.kubeconfig), "MCP_PORT=" + str(port),
                    "MCP_HEALTH_TIMEOUT=10", "MCP_HEALTH_INTERVAL=1", "TOOLSETS=core,config", "READ_ONLY=true",
                ], cwd=working, env=environment, stdout=output, stderr=subprocess.STDOUT, timeout=30)
            finally:
                pidfile = working / ".mcp-server.pid"
                if pidfile.exists():
                    self.stop_server(int(pidfile.read_text()))
            output.seek(0)
            log = output.read().decode()
        self.assertEqual(0, result.returncode, log)
        self.assertIn("MCP server is ready", log)
        for original in source.glob("*.toml"):
            self.assertEqual(original.read_bytes(), (generated / original.name).read_bytes())
        self.assertEqual('read_only = true\ntoolsets = ["core", "config"]\n', (generated / "zz-eval-overrides.toml").read_text())

    def test_each_observability_insecure_flag_stays_rejected(self):
        source = (self.repository / "dev/config/mcp-configs/obs.toml").read_text()
        sections = re.split(r'(?m)(?=^\[toolset_configs\.)', source)[1:]
        self.assertEqual(3, len(sections))
        for name, section in zip(("metrics", "traces", "logs"), sections):
            with self.subTest(toolset=name):
                self.assertTrue(section.startswith('[toolset_configs."observability/%s"]' % name))
                section = re.sub(r"(?m)^insecure = true\n", "", section)
                section = section.replace("\n", "\ninsecure = true\n", 1)
                config = self.root / (name + ".toml")
                config.write_text(self.configuration() + section)
                result = subprocess.run([str(self.server), "--config", str(config)], stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=15)
                self.assertEqual(1, result.returncode, result.stderr)
                self.assertIn("insecure=true is no longer supported", result.stderr)

    def test_require_tls_still_rejects_http_loopback(self):
        config = self.root / "require-tls.toml"
        source = (self.repository / "dev/config/mcp-configs/obs.toml").read_text()
        config.write_text(self.configuration() + "require_tls = true\n" + re.sub(r"(?m)^insecure = true\n", "", source))
        result = subprocess.run([str(self.server), "--config", str(config)], stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=15)
        self.assertEqual(1, result.returncode, result.stderr)
        self.assertIn("require_tls is enabled", result.stderr)


if __name__ == "__main__":
    unittest.main()
