import importlib.util
import unittest
from pathlib import Path

_SPEC = importlib.util.spec_from_file_location(
    "check_docstrings", Path(__file__).parent.parent / "scripts" / "check_docstrings.py"
)
cd = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(cd)

COMPLETE = '''
def f():
    """Summary.

    Algorithm: a.
    Techniques: b.
    Keywords: c.
    References: https://example.com
    """
'''


class CheckDocstringsTest(unittest.TestCase):
    def test_flags_function_without_sections(self):
        got = cd.check_functions('def f():\n    """One line."""\n')
        self.assertEqual(
            got,
            [f"f: missing {s}" for s in ("Algorithm", "Techniques", "Keywords", "References")],
        )

    def test_accepts_complete_docstring(self):
        self.assertEqual(cd.check_functions(COMPLETE), [])

    def test_flags_missing_docstring_and_skips_dunder_and_nested(self):
        src = (
            "class A:\n    def __init__(self): pass\n"
            "    def m(self):\n        def inner(): pass\n"
        )
        self.assertEqual(len(cd.check_functions(src)), 4)
        self.assertTrue(all(x.startswith("m:") for x in cd.check_functions(src)))

    def test_check_links_uses_injected_fetch(self):
        self.assertEqual(
            cd.check_links(["u1", "u2"], fetch=lambda u: 200 if u == "u1" else 404),
            {"u1": 200, "u2": 404},
        )

    def test_extract_urls(self):
        self.assertEqual(cd.extract_urls(COMPLETE), ["https://example.com"])


if __name__ == "__main__":
    unittest.main()
