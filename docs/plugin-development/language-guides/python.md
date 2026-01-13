# Python Plugin Development

This guide covers Python-specific best practices for Specular plugin development.

## Why Python?

- **Rapid Development:** Quick prototyping and iteration
- **Rich Ecosystem:** Extensive libraries for ML, HTTP, data processing
- **Readability:** Clean, maintainable code
- **AI/ML Integration:** Native support for AI libraries

## Quick Start

```bash
specular plugin create my-plugin --type validator --lang python
cd my-plugin
echo '{"action":"health"}' | python3 main.py
```

## Project Structure

```
my-plugin/
├── plugin.yaml       # Plugin manifest
├── main.py           # Entry point
├── requirements.txt  # Dependencies
├── handler.py        # Action handlers
├── config.py         # Configuration
├── utils/
│   └── helpers.py    # Utilities
└── tests/
    ├── test_handler.py
    └── test_config.py
```

## Code Templates

### Basic Plugin Structure

```python
#!/usr/bin/env python3
"""My Plugin - A Specular plugin."""

import json
import sys
from typing import Any, Dict, Optional

VERSION = "1.0.0"


def handle_request(request: Dict[str, Any]) -> Dict[str, Any]:
    """Handle incoming plugin request."""
    action = request.get("action", "")

    handlers = {
        "health": handle_health,
        "validate": handle_validate,
    }

    handler = handlers.get(action)
    if handler is None:
        return {
            "success": False,
            "error": f"unknown action: {action}",
        }

    try:
        result = handler(request)
        return {"success": True, "result": result}
    except PluginError as e:
        return {"success": False, "error": str(e)}
    except Exception as e:
        return {"success": False, "error": f"internal: {e}"}


def handle_health(request: Dict[str, Any]) -> Dict[str, str]:
    """Handle health check."""
    return {
        "status": "healthy",
        "version": VERSION,
    }


def handle_validate(request: Dict[str, Any]) -> Dict[str, Any]:
    """Handle validation request."""
    data = request.get("data", {})
    config = request.get("config", {})

    # Your validation logic here
    content = data.get("content", "")

    return {
        "valid": True,
        "messages": [],
    }


def main() -> None:
    """Main entry point."""
    for line in sys.stdin:
        try:
            request = json.loads(line.strip())
            response = handle_request(request)
        except json.JSONDecodeError as e:
            response = {"success": False, "error": f"json: {e}"}

        print(json.dumps(response), flush=True)


if __name__ == "__main__":
    main()
```

### Custom Error Types

```python
class PluginError(Exception):
    """Base exception for plugin errors."""

    def __init__(self, category: str, message: str):
        self.category = category
        self.message = message
        super().__init__(f"{category}: {message}")


class ConfigError(PluginError):
    """Configuration error."""

    def __init__(self, message: str):
        super().__init__("config", message)


class ValidationError(PluginError):
    """Validation error."""

    def __init__(self, message: str):
        super().__init__("validation", message)


class NetworkError(PluginError):
    """Network error."""

    def __init__(self, message: str):
        super().__init__("network", message)
```

### Type-Safe Configuration

```python
from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class Config:
    """Plugin configuration."""

    api_key: str
    endpoint: str = "https://api.example.com"
    timeout: int = 30
    retries: int = 3
    debug: bool = False

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Config":
        """Create Config from dictionary."""
        api_key = data.get("api_key")
        if not api_key:
            raise ConfigError("api_key is required")

        return cls(
            api_key=api_key,
            endpoint=data.get("endpoint", cls.endpoint),
            timeout=int(data.get("timeout", cls.timeout)),
            retries=int(data.get("retries", cls.retries)),
            debug=bool(data.get("debug", cls.debug)),
        )

    def validate(self) -> None:
        """Validate configuration."""
        if self.timeout <= 0:
            raise ConfigError("timeout must be positive")
        if self.retries < 0:
            raise ConfigError("retries cannot be negative")
```

### HTTP Client with Retry

```python
import time
from typing import Any, Dict, Optional
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry


class APIClient:
    """HTTP client with retry support."""

    def __init__(self, config: Config):
        self.config = config
        self.session = self._create_session()

    def _create_session(self) -> requests.Session:
        """Create session with retry configuration."""
        session = requests.Session()

        retry = Retry(
            total=self.config.retries,
            backoff_factor=0.5,
            status_forcelist=[500, 502, 503, 504],
            allowed_methods=["GET", "POST", "PUT", "DELETE"],
        )

        adapter = HTTPAdapter(max_retries=retry)
        session.mount("http://", adapter)
        session.mount("https://", adapter)

        session.headers.update({
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
        })

        return session

    def request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make HTTP request."""
        url = f"{self.config.endpoint}{path}"

        try:
            response = self.session.request(
                method,
                url,
                json=data,
                timeout=self.config.timeout,
            )
            response.raise_for_status()
            return response.json()
        except requests.Timeout:
            raise NetworkError(f"request timeout after {self.config.timeout}s")
        except requests.ConnectionError as e:
            raise NetworkError(f"connection failed: {e}")
        except requests.HTTPError as e:
            raise NetworkError(f"HTTP error: {e.response.status_code}")

    def close(self) -> None:
        """Close the session."""
        self.session.close()
```

### Async Support

```python
import asyncio
import aiohttp
from typing import Any, Dict, Optional


class AsyncAPIClient:
    """Async HTTP client."""

    def __init__(self, config: Config):
        self.config = config
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        """Get or create session."""
        if self._session is None or self._session.closed:
            timeout = aiohttp.ClientTimeout(total=self.config.timeout)
            self._session = aiohttp.ClientSession(
                timeout=timeout,
                headers={
                    "Authorization": f"Bearer {self.config.api_key}",
                    "Content-Type": "application/json",
                },
            )
        return self._session

    async def request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make async HTTP request."""
        session = await self._get_session()
        url = f"{self.config.endpoint}{path}"

        try:
            async with session.request(method, url, json=data) as response:
                response.raise_for_status()
                return await response.json()
        except asyncio.TimeoutError:
            raise NetworkError("request timeout")
        except aiohttp.ClientError as e:
            raise NetworkError(str(e))

    async def close(self) -> None:
        """Close the session."""
        if self._session:
            await self._session.close()


# Usage with async handler
async def handle_notify_async(request: Dict[str, Any]) -> Dict[str, Any]:
    """Handle notification with async client."""
    config = Config.from_dict(request.get("config", {}))
    client = AsyncAPIClient(config)

    try:
        result = await client.request("POST", "/notify", request.get("data", {}))
        return {"delivered": True, "response": result}
    finally:
        await client.close()


def handle_notify(request: Dict[str, Any]) -> Dict[str, Any]:
    """Sync wrapper for async handler."""
    return asyncio.run(handle_notify_async(request))
```

### Logging

```python
import logging
import sys
from typing import Optional


def setup_logger(name: str, debug: bool = False) -> logging.Logger:
    """Configure logger to write to stderr."""
    logger = logging.getLogger(name)
    logger.setLevel(logging.DEBUG if debug else logging.INFO)

    # Only log to stderr, never stdout
    handler = logging.StreamHandler(sys.stderr)
    handler.setFormatter(logging.Formatter(
        "[%(levelname)s] %(name)s: %(message)s"
    ))

    logger.addHandler(handler)
    return logger


# Global logger
logger = setup_logger("my-plugin")


# Usage
def handle_validate(request: Dict[str, Any]) -> Dict[str, Any]:
    logger.debug(f"Validating request: {request.get('action')}")

    try:
        # Validation logic
        logger.info("Validation successful")
        return {"valid": True}
    except Exception as e:
        logger.error(f"Validation failed: {e}")
        raise
```

### Input Validation with Pydantic

```python
from pydantic import BaseModel, Field, validator
from typing import Any, Dict, List, Optional


class ValidationRequest(BaseModel):
    """Validated request model."""

    content: str = Field(..., min_length=1)
    rules: List[str] = Field(default_factory=list)
    severity_threshold: str = Field(default="warning")

    @validator("severity_threshold")
    def validate_severity(cls, v: str) -> str:
        valid = {"error", "warning", "info"}
        if v not in valid:
            raise ValueError(f"severity must be one of {valid}")
        return v

    @classmethod
    def from_request(cls, data: Dict[str, Any]) -> "ValidationRequest":
        """Create from request data."""
        return cls(**data)


# Usage
def handle_validate(request: Dict[str, Any]) -> Dict[str, Any]:
    try:
        req = ValidationRequest.from_request(request.get("data", {}))
    except ValueError as e:
        raise ValidationError(str(e))

    # Use validated req.content, req.rules, etc.
    return {"valid": True}
```

## Testing

### Unit Tests with pytest

```python
# tests/test_handler.py
import pytest
from main import handle_request, handle_health, handle_validate


def test_handle_health():
    """Test health check."""
    result = handle_health({})
    assert result["status"] == "healthy"
    assert "version" in result


def test_handle_request_health():
    """Test request routing for health."""
    response = handle_request({"action": "health"})
    assert response["success"] is True
    assert response["result"]["status"] == "healthy"


def test_handle_request_unknown_action():
    """Test unknown action error."""
    response = handle_request({"action": "invalid"})
    assert response["success"] is False
    assert "unknown action" in response["error"]


class TestValidate:
    """Validation tests."""

    def test_valid_content(self):
        """Test valid content passes."""
        request = {
            "action": "validate",
            "data": {"content": "valid content"},
        }
        response = handle_request(request)
        assert response["success"] is True
        assert response["result"]["valid"] is True

    def test_missing_content(self):
        """Test missing content fails."""
        request = {
            "action": "validate",
            "data": {},
        }
        response = handle_request(request)
        assert response["success"] is False


# Fixtures
@pytest.fixture
def sample_config():
    """Sample configuration."""
    return {
        "api_key": "test-key",
        "endpoint": "https://api.test.com",
    }
```

### Integration Tests

```python
# tests/test_integration.py
import json
import subprocess
import pytest


@pytest.fixture
def plugin_process():
    """Start plugin process."""
    proc = subprocess.Popen(
        ["python3", "main.py"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    yield proc
    proc.terminate()


def send_request(proc, request: dict) -> dict:
    """Send request and get response."""
    proc.stdin.write(json.dumps(request) + "\n")
    proc.stdin.flush()
    response = proc.stdout.readline()
    return json.loads(response)


def test_plugin_health(plugin_process):
    """Test health check via process."""
    response = send_request(plugin_process, {"action": "health"})
    assert response["success"] is True


def test_plugin_validate(plugin_process):
    """Test validation via process."""
    request = {
        "action": "validate",
        "data": {"content": "test content"},
    }
    response = send_request(plugin_process, request)
    assert response["success"] is True
```

### Mocking External Services

```python
import pytest
from unittest.mock import Mock, patch
from main import handle_notify


@pytest.fixture
def mock_requests():
    """Mock requests library."""
    with patch("main.requests") as mock:
        mock.post.return_value = Mock(
            ok=True,
            json=Mock(return_value={"message_id": "123"})
        )
        yield mock


def test_notify_success(mock_requests):
    """Test successful notification."""
    request = {
        "data": {"message": "test"},
        "config": {"webhook_url": "https://example.com/hook"},
    }
    result = handle_notify(request)
    assert result["delivered"] is True
    mock_requests.post.assert_called_once()


def test_notify_failure(mock_requests):
    """Test notification failure."""
    mock_requests.post.side_effect = Exception("Connection failed")

    request = {
        "data": {"message": "test"},
        "config": {"webhook_url": "https://example.com/hook"},
    }

    with pytest.raises(NetworkError):
        handle_notify(request)
```

## Dependencies

### requirements.txt

```
# Core
requests>=2.28.0
pydantic>=2.0.0

# Async support (optional)
aiohttp>=3.8.0

# Testing
pytest>=7.0.0
pytest-asyncio>=0.21.0

# Development
black>=23.0.0
mypy>=1.0.0
flake8>=6.0.0
```

### Virtual Environment

```bash
# Create environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Install for development
pip install -e ".[dev]"
```

## Type Checking with mypy

```python
# mypy.ini
[mypy]
python_version = 3.9
strict = true
warn_return_any = true
warn_unused_configs = true

[mypy-requests.*]
ignore_missing_imports = true
```

```bash
# Run type checker
mypy main.py
```

## Performance Tips

1. **Use generators** for large data processing
2. **Cache HTTP sessions** across requests
3. **Use async I/O** for network-heavy plugins
4. **Profile with cProfile** to find bottlenecks

```python
# Generator for large datasets
def process_items(items):
    for item in items:
        yield process_single(item)

# Cache session
_session = None

def get_session():
    global _session
    if _session is None:
        _session = requests.Session()
    return _session
```

## Common Pitfalls

1. **Forgetting to flush stdout** - Always use `flush=True` with `print()`
2. **Logging to stdout** - All logs must go to stderr
3. **Not handling JSON errors** - Wrap JSON parsing in try/except
4. **Blocking on I/O** - Use timeouts for all network calls
5. **Memory leaks** - Close sessions and file handles
