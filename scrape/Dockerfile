FROM python:3.13-slim

RUN apt-get update && apt-get install -y \
    git \
    curl \
    build-essential \
    ca-certificates \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && rm -rf /var/lib/apt/lists/*

RUN pip install uv

WORKDIR /app

COPY api/clients/python/ /api/clients/python/

COPY scrape/pyproject.toml scrape/uv.lock ./
COPY scrape/packages/ packages/
COPY scrape/src/ src/
COPY scrape/data/ data/

RUN uv venv && uv sync --frozen

RUN uv run python -m spacy download en_core_web_lg

ENV PATH="/app/.venv/bin:$PATH"

EXPOSE 3000

CMD ["uv", "run", "dagster", "dev", "--host", "0.0.0.0", "--port", "3000"]
