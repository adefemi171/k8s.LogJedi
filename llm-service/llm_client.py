"""Abstract LLM client - mock and placeholder for real provider."""

from typing import Optional


class BaseLLMClient:
    """Interface for LLM completion. Implement mock and real providers."""

    async def complete(self, system_prompt: str, user_prompt: str) -> str:
        """Return completion text. Override in implementations."""
        raise NotImplementedError


class MockLLMClient(BaseLLMClient):
    """Returns fixed response for local testing."""

    async def complete(self, system_prompt: str, user_prompt: str) -> str:
        return (
            "Summary: Mock analysis. Root cause: N/A. "
            "Recommendation: Check cluster events and pod logs."
        )


def get_llm_client(provider: str, api_key: Optional[str] = None) -> BaseLLMClient:
    """Factory: returns mock or real client based on LLM_PROVIDER."""
    if provider == "mock" or not provider:
        return MockLLMClient()
    # Placeholder for openai etc. - config-driven, no hardcoded provider
    return MockLLMClient()
