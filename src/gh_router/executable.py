from __future__ import annotations

import os
import shutil
from pathlib import Path
from typing import Optional


class ExecutableError(Exception):
    pass


def find_real_gh(argv0: Optional[str] = None, environ: Optional[dict[str, str]] = None) -> str:
    environment = environ or os.environ
    configured = environment.get("GH_ROUTER_REAL_GH")
    if configured:
        path = shutil.which(configured) or configured
        if Path(path).exists():
            return path
        raise ExecutableError("GH_ROUTER_REAL_GH does not point to an executable: {}".format(configured))

    router_path = Path(argv0 or "").resolve() if argv0 else None
    for directory in environment.get("PATH", "").split(os.pathsep):
        if not directory:
            continue
        candidate = Path(directory) / "gh"
        if not candidate.is_file() or not os.access(candidate, os.X_OK):
            continue
        if router_path and candidate.resolve() == router_path:
            continue
        return str(candidate)

    for candidate in ("/opt/homebrew/bin/gh", "/usr/local/bin/gh", "/usr/bin/gh"):
        path = Path(candidate)
        if path.is_file() and os.access(path, os.X_OK):
            if not router_path or path.resolve() != router_path:
                return candidate
    raise ExecutableError("Unable to find the real gh executable; set GH_ROUTER_REAL_GH")
