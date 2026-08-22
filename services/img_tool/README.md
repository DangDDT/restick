# img_tool

Automated image processing worker, sticker sheet slicer, and standalone CLI utility for Restick.

## 🎯 Responsibilities

- **Sticker Sheet Slicing**: Automatically segments and extracts individual sticker elements from composite sticker sheets using OpenCV alpha thresholding and contour hierarchy analysis.
- **Normalization Engine**: Standardizes arbitrary sticker dimensions to the 512x512 transparent WebP format required by mobile messaging applications.
- **Microservice Worker**: Connects to `api_svc` and `ftp_svc` to process background conversion jobs asynchronously via gRPC.
- **Developer CLI**: Provides a standalone command-line tool for designers and developers to slice and optimize sticker sheets locally.

---

## ⚙️ Configuration (Environment Variables)

| Variable | Default | Description |
| :--- | :--- | :--- |
| `IMG_HTTP_ADDR` | `:8080` | Bind address for HTTP health check |
| `IMG_GRPC_ADDR` | `:50053` | Bind address for gRPC worker server |
| `IMG_TOOL_ENGINE` | `simple` | Processing engine algorithm (`simple` or `contour`) |
| `PUBLIC_FTP_ORIGIN` | `http://localhost:8081` | Base URL used to construct downloadable asset links |
| `API_SVC_GRPC_ADDR` | `api_svc:50051` | gRPC address of `api_svc` |
| `FTP_SVC_GRPC_ADDR` | `ftp_svc:50052` | gRPC address of `ftp_svc` |

---

## 💻 CLI Usage

Install `img_tool` locally using `pip`:

```bash
cd services/img_tool
pip install -e .
```

### Slicing a Sticker Sheet
```bash
img_tool slice input_sheet.png --out-dir ./output_stickers --padding 16
```

### Normalizing a Single Sticker
```bash
img_tool normalize sticker_raw.png --out ./sticker_512.webp --size 512
```

---

## 🚀 Running the Microservice

### With Python
```bash
cd services/img_tool
python -m img_tool.main
```

### With Docker
```bash
docker compose -f docker-compose.yml up --build
```
