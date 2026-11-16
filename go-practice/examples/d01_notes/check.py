#!/usr/bin/env python3
"""Run the exercise contracts against the reference in an isolated copy."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    example = Path(__file__).resolve().parent
    project = example.parents[1]
    relative = Path("04_crud/d01_notes")
    if not (os.environ.get("TEST_DATABASE_URL") or os.environ.get("DATABASE_URL")):
        raise SystemExit("Set TEST_DATABASE_URL for the local practice PostgreSQL on port 15432.")

    implementations = (
        "handler/handler.go",
        "cmd/server/main.go",
        "schema.sql",
    )
    with tempfile.TemporaryDirectory(prefix="notes-reference-") as directory:
        temporary = Path(directory)
        for name in ("go.mod", "go.sum"):
            shutil.copy2(project / name, temporary / name)
        shutil.copytree(project / relative, temporary / relative)
        for name in implementations:
            source = (example / name).read_text(encoding="utf-8")
            source = source.replace(
                "example.com/go-practice/examples/d01_notes",
                "example.com/go-practice/04_crud/d01_notes",
            )
            (temporary / relative / name).write_text(source, encoding="utf-8")
        for command in (
            ["go", "test", "-race", "-p=1", "-count=1", "-timeout=60s", "./04_crud/d01_notes/..."],
            ["go", "vet", "./04_crud/d01_notes/..."],
        ):
            result = subprocess.run(command, cwd=temporary, check=False)
            if result.returncode:
                raise SystemExit(result.returncode)


if __name__ == "__main__":
    main()
