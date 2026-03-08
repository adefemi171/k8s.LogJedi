"""FastAPI application for k8s LogJedi LLM analysis service."""

from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI

from config import HOST, PORT
from routers import analyze, health

@asynccontextmanager
async def lifespan(app: FastAPI):
    yield
    # shutdown cleanup if needed


app = FastAPI(
    title="k8s LogJedi LLM Service",
    description="Analyzes failed Kubernetes workloads and returns explanations and optional patches.",
    version="0.1.0",
    lifespan=lifespan,
)
app.include_router(analyze.router, prefix="")
app.include_router(health.router, prefix="")


if __name__ == "__main__":
    uvicorn.run("main:app", host=HOST, port=PORT, reload=True)
