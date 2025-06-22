FROM python:3.13-slim

RUN apt-get update && apt-get install -y \
    git \
    curl \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN pip install uv

WORKDIR /app

COPY pyproject.toml uv.lock ./

COPY packages/ packages/

COPY src/ src/

RUN uv sync --frozen

COPY dagster.yaml ./

EXPOSE 3000

CMD ["uv", "run", "dagster", "dev", "--host", "0.0.0.0", "--port", "3000"]
