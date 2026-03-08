FROM python:3.14-slim

WORKDIR /app

# Use uv for fast, reproducible installs (binary at root in distroless image)
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

COPY pyproject.toml requirements.txt ./
RUN uv pip install --system --no-cache -r requirements.txt
COPY . .

EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
