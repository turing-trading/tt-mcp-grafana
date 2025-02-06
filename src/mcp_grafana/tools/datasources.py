from mcp.server import FastMCP

from ..client import GrafanaClient


async def list_datasources() -> bytes:
    """
    List datasources in the Grafana instance.
    """
    return await GrafanaClient.for_current_request().list_datasources()


async def get_datasource_by_uid(uid: str) -> bytes:
    """
    Get a datasource by uid.
    """
    return await GrafanaClient.for_current_request().get_datasource(uid=uid)


async def get_datasource_by_name(name: str) -> bytes:
    """
    Get a datasource by name.
    """
    return await GrafanaClient.for_current_request().get_datasource(name=name)


def add_tools(mcp: FastMCP):
    mcp.add_tool(list_datasources)
    mcp.add_tool(get_datasource_by_uid)
    mcp.add_tool(get_datasource_by_name)
