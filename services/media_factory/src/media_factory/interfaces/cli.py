from __future__ import annotations

from pathlib import Path
from uuid import uuid4

import typer

from media_factory.application.process_sheet import execute_process

app: typer.Typer = typer.Typer(
    help="Restick image pipeline. Does not generate images from keywords.",
    no_args_is_help=True,
)


@app.command("version")
def version_command() -> None:
    """In phiên bản của công cụ.

    Algorithm:
        1. Ghi '0.0.1' ra stdout.

    Techniques:
        - CLI command (Typer)

    Keywords: version, CLI, typer

    References:
        - https://en.wikipedia.org/wiki/Command-line_interface
    """
    typer.echo("0.0.1")


@app.command("process")
def process_command(
    sheet: Path | None = typer.Argument(None, exists=True, readable=True),
    files: str = typer.Option("", help="Comma-separated already-cut image paths"),
    name: str = typer.Option(..., "--name"),
    slug: str = typer.Option(..., "--slug"),
    author: str = typer.Option("Restick", "--author"),
    tags: str = typer.Option("", "--tags"),
    out: Path = typer.Option(..., "--out"),
    min_area: int = typer.Option(800, "--min-area"),
    padding: int = typer.Option(12, "--padding"),
    lossless: bool = typer.Option(True, "--lossless/--lossy"),
) -> None:
    """Lệnh CLI 'process': dựng pack từ sheet hoặc danh sách file.

    Algorithm:
        1. Tách --files theo dấu phẩy; mỗi file phải tồn tại.
        2. Sinh pack_id (uuid4), tách --tags.
        3. Gọi execute_process và in đường dẫn cùng số sticker.

    Techniques:
        - CLI argument parsing (Typer)
        - Input validation

    Keywords: CLI, typer, process, pack, arguments

    References:
        - https://en.wikipedia.org/wiki/Command-line_interface
        - https://en.wikipedia.org/wiki/Input_validation
    """
    file_paths: list[Path] = []
    if files.strip() != "":
        file_paths = [Path(part.strip()) for part in files.split(",") if part.strip() != ""]
        for path in file_paths:
            if not path.is_file():
                raise typer.BadParameter(f"missing file {path}")
    pack_id = str(uuid4())
    pack_out = out / pack_id
    tag_list = [part.strip() for part in tags.split(",") if part.strip() != ""]
    pack = execute_process(
        pack_id=pack_id,
        name=name,
        slug=slug,
        author=author,
        tags=tag_list,
        out_dir=pack_out,
        sheet=sheet,
        files=file_paths,
        min_area=min_area,
        padding=padding,
        lossless=lossless,
    )
    typer.echo(f"wrote {pack_out} stickers={len(pack.stickers)}")


def main() -> None:
    """Điểm vào của CLI.

    Algorithm:
        1. Chạy ứng dụng Typer.

    Techniques:
        - CLI entrypoint

    Keywords: entrypoint, typer, main

    References:
        - https://en.wikipedia.org/wiki/Command-line_interface
    """
    app()


if __name__ == "__main__":
    main()
