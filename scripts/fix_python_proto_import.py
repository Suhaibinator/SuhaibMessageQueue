"""Make grpcio-tools output importable from the pysmq package."""

from pathlib import Path

generated_file = (
    Path(__file__).resolve().parents[1]
    / "python"
    / "src"
    / "pysmq"
    / "_generated"
    / "smq_pb2_grpc.py"
)
content = generated_file.read_text(encoding="utf-8")
absolute_import = "import smq_pb2 as smq__pb2"
relative_import = "from . import smq_pb2 as smq__pb2"
if absolute_import not in content and relative_import not in content:
    raise SystemExit(f"expected generated import not found in {generated_file}")
generated_file.write_text(content.replace(absolute_import, relative_import), encoding="utf-8")
