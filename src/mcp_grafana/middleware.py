from dataclasses import dataclass

from starlette.datastructures import Headers

from .client import GrafanaClient, grafana_client
from .settings import GrafanaSettings, grafana_settings


@dataclass
class GrafanaInfo:
    """
    Simple container for the Grafana URL and API key.
    """

    authorization: str
    url: str

    @classmethod
    def from_headers(cls, headers: Headers) -> "GrafanaInfo | None":
        if (url := headers.get("X-Grafana-URL")) is not None and (
            key := headers.get("X-Grafana-API-Key")
        ) is not None:
            return cls(authorization=key, url=url)
        return None


class GrafanaMiddleware:
    """
    Middleware that sets up Grafana info for the current request.

    Grafana info will be stored in the `grafana_info` contextvar, which can be
    used by tools/resources etc to access the Grafana configuration for the
    current request, if it was provided.

    This should be used as a context manager before handling the /sse request.
    """

    def __init__(self, request):
        self.request = request
        self.token = None

    async def __aenter__(self):
        if (info := GrafanaInfo.from_headers(self.request.headers)) is not None:
            current_settings = grafana_settings.get()
            new_settings = GrafanaSettings(
                url=info.url,
                api_key=info.authorization,
                tools=current_settings.tools,
            )
            self.settings_token = grafana_settings.set(new_settings)
            self.client_token = grafana_client.set(
                GrafanaClient.from_settings(new_settings)
            )

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.token is not None:
            grafana_settings.reset(self.token)
