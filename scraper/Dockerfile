FROM python:3.13-slim

RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r appuser && useradd -r -g appuser appuser

RUN pip install pipenv

WORKDIR /app

COPY Pipfile Pipfile.lock ./

RUN pipenv install --system --deploy
RUN python -m spacy download en_core_web_lg

COPY . .

RUN chown -R appuser:appuser /app
USER appuser

ENTRYPOINT ["python", "main.py"]
