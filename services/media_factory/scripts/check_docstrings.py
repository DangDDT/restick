"""Manual docstring checker: four sections per function, and every link returns HTTP 200.

Usage: python scripts/check_docstrings.py src/media_factory
"""

import ast
import re
import subprocess
import sys
import urllib.request
from collections.abc import Callable, Iterable
from pathlib import Path

SECTIONS = ("Algorithm", "Techniques", "Keywords", "References")
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120 Safari/537.36"
URL_RE = re.compile(r"https?://\S+")


def _functions(tree: ast.AST):
    """Yield module-level functions and class methods; nested functions are skipped."""
    for node in ast.iter_child_nodes(tree):
        if isinstance(node, ast.ClassDef):
            yield from _functions(node)
        elif isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
            yield node


def check_functions(source: str) -> list[str]:
    problems = []
    for fn in _functions(ast.parse(source)):
        if fn.name.startswith("__") and fn.name.endswith("__"):
            continue
        doc = ast.get_docstring(fn) or ""
        problems += [f"{fn.name}: missing {s}" for s in SECTIONS if f"{s}:" not in doc]
    return problems


def extract_urls(source: str) -> list[str]:
    urls = []
    for fn in _functions(ast.parse(source)):
        urls += [u.rstrip(".,;") for u in URL_RE.findall(ast.get_docstring(fn) or "")]
    return urls


def _http_status(url: str) -> int:
    status = _try(url)
    if status in (403, 429):  # Cloudflare challenges urllib's TLS fingerprint; curl passes it
        try:
            out = subprocess.run(
                ["curl", "-s", "-o", "/dev/null", "-L", "-A", UA, "-m", "15",
                 "-w", "%{http_code}", url],
                capture_output=True,
                text=True,
                timeout=20,
            ).stdout
            status = int(out) or status
        except (OSError, ValueError, subprocess.TimeoutExpired):
            pass
    return status


def _try(url: str) -> int:
    for method in ("HEAD", "GET"):
        headers = {"User-Agent": UA, "Accept": "*/*"}
        req = urllib.request.Request(url, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=15) as r:
                return r.status
        except urllib.error.HTTPError as e:
            status = e.code
        except OSError:
            status = 0
    return status


def check_links(
    urls: Iterable[str], fetch: Callable[[str], int] = _http_status
) -> dict[str, int]:
    return {u: fetch(u) for u in dict.fromkeys(urls)}


def main(root: str) -> int:
    problems, urls = [], []
    for path in sorted(Path(root).rglob("*.py")):
        src = path.read_text()
        problems += [f"{path}: {p}" for p in check_functions(src)]
        urls += extract_urls(src)
    statuses = check_links(urls)
    problems += [f"{u}: HTTP {s}" for u, s in statuses.items() if s != 200]
    print("\n".join(problems) or "OK")
    print(f"links checked: {len(statuses)}, non-200: {sum(s != 200 for s in statuses.values())}")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
