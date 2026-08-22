from __future__ import annotations

import logging
from pathlib import Path
from uuid import uuid4

import typer

from img_tool.application.process_sheet import execute_process
from img_tool.application.submit_pack import execute_submit
from img_tool.infrastructure.config.config import Config
from img_tool.infrastructure.grpc_clients import JobClient, ObjectClient
from img_tool.interfaces.serve import serve as run_serve
from img_tool.interfaces.worker import run_worker

app: typer.Typer = typer.Typer(
    help="Restick image pipeline. Does not generate images from keywords.",
    no_args_is_help=True,
)


@app.command("version")
def version_command() -> None:
    typer.echo("0.0.1")


@app.command("serve")
def serve_command() -> None:
    run_serve()


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


@app.command("submit")
def submit_command(out_dir: Path = typer.Argument(..., exists=True, file_okay=False)) -> None:
    cfg = Config()
    if cfg.ftp_svc_grpc_addr == "" or cfg.api_svc_grpc_addr == "":
        raise typer.BadParameter("FTP_SVC_GRPC_ADDR and API_SVC_GRPC_ADDR are required")
    objects = ObjectClient(cfg.ftp_svc_grpc_addr, cfg.internal_token)
    jobs = JobClient(cfg.api_svc_grpc_addr, cfg.internal_token)
    execute_submit(out_dir, objects, jobs)
    typer.echo("submitted pending_review")


@app.command("worker")
def worker_command() -> None:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    run_worker(Config())


def main() -> None:
    app()


if __name__ == "__main__":
    main()
