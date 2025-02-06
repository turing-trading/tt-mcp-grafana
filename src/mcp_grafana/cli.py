import enum
from types import MethodType

import typer

from . import mcp

app = typer.Typer()


class Transport(enum.StrEnum):
    stdio = "stdio"
    sse = "sse"


@app.command()
def run(transport: Transport = Transport.stdio, header_auth: bool = False):
    if transport == Transport.sse and header_auth:
        # Monkeypatch the run_sse_async method to inject a Grafana middleware.
        # This is a bit of a hack, but fastmcp doesn't have a way of adding
        # middleware. It's not unreasonable to do this really, since fastmcp
        # is just a thin wrapper around the low level mcp server.
        mcp.run_sse_async = MethodType(run_sse_async, mcp)

    mcp.run(transport.value)


async def run_sse_async(self) -> None:
    """
    Run the server using SSE transport, with a middleware that extracts
    Grafana authentication information from the request headers.

    The vast majority of this code is the same as the original run_sse_async
    method (see https://github.com/modelcontextprotocol/python-sdk/blob/44c0004e6c69e336811bb6793b7176e1eda50015/src/mcp/server/fastmcp/server.py#L436-L468).
    """

    from mcp.server.sse import SseServerTransport
    from starlette.applications import Starlette
    from starlette.routing import Mount, Route
    import uvicorn

    from .middleware import GrafanaMiddleware

    sse = SseServerTransport("/messages/")

    async def handle_sse(request):
        async with GrafanaMiddleware(request):
            async with sse.connect_sse(
                request.scope, request.receive, request._send
            ) as streams:
                await self._mcp_server.run(
                    streams[0],
                    streams[1],
                    self._mcp_server.create_initialization_options(),
                )

    starlette_app = Starlette(
        debug=self.settings.debug,
        routes=[
            Route("/sse", endpoint=handle_sse),
            Mount("/messages/", app=sse.handle_post_message),
        ],
    )

    config = uvicorn.Config(
        starlette_app,
        host=self.settings.host,
        port=self.settings.port,
        log_level=self.settings.log_level.lower(),
    )
    server = uvicorn.Server(config)
    await server.serve()
