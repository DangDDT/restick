FROM python:3.12-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl libglib2.0-0 libgomp1 \
    && rm -rf /var/lib/apt/lists/*
COPY services/img_tool/pyproject.toml /app/img_tool/pyproject.toml
COPY services/img_tool/src /app/img_tool/src
COPY contracts/gen/python /opt/restick/gen
ENV PYTHONPATH=/app/img_tool/src:/opt/restick/gen
RUN pip install --no-cache-dir /app/img_tool
EXPOSE 8080 50053
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
CMD ["img_tool", "serve"]
