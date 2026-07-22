import os
import signal
import socket
import subprocess
import time

import httpx
import pytest


def _add_no_proxy(*hosts):
    current = os.environ.get("NO_PROXY") or os.environ.get("no_proxy") or ""
    entries = [p.strip() for p in current.split(",") if p.strip()]
    for host in hosts:
        if host not in entries:
            entries.append(host)
    os.environ["NO_PROXY"] = ",".join(entries)
    os.environ["no_proxy"] = os.environ["NO_PROXY"]


def _free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


_add_no_proxy("localhost", "127.0.0.1", "::1")

if "BACKEND_URL" not in os.environ:
    port = str(_free_port())
    os.environ["BACKEND_PORT"] = port
    os.environ["BACKEND_URL"] = f"http://localhost:{port}"

BACKEND_URL = os.environ["BACKEND_URL"]


@pytest.fixture(scope="session", autouse=True)
def backend_server():
    backend_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "backend")
    proc = subprocess.Popen(
        ["go", "run", "./cmd/server"],
        cwd=backend_dir,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        preexec_fn=os.setsid,
    )
    for _ in range(30):
        if proc.poll() is not None:
            stdout, stderr = proc.communicate()
            raise RuntimeError(
                f"Backend failed to start\nstdout={stdout.decode()}\nstderr={stderr.decode()}"
            )
        try:
            r = httpx.get(f"{BACKEND_URL}/api/health", timeout=1, trust_env=False)
            if r.status_code == 200:
                break
        except Exception:
            time.sleep(0.5)
    else:
        os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
        proc.wait(timeout=5)
        raise RuntimeError("Backend failed to start")

    yield proc

    os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
    proc.wait(timeout=5)
