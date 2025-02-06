from mcp.server import FastMCP

from ..client import SearchDashboardsArguments, GrafanaClient


async def search_dashboards(arguments: SearchDashboardsArguments) -> bytes:
    """
    Search dashboards in the Grafana instance.
    """
    return await GrafanaClient.for_current_request().search_dashboards(arguments)


def add_tools(mcp: FastMCP):
    mcp.add_tool(search_dashboards)
