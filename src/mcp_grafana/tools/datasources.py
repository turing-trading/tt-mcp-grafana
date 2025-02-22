from typing import List
from mcp.server import FastMCP

from mcp_grafana.grafana_types import Datasource

from ..client import grafana_client


async def list_datasources() -> List[Datasource]:
    """
    List datasources in the Grafana instance.
    """
    datasources = await grafana_client.list_datasources()
    resp = []
    # Only push a subset of fields to save on space.
    for ds in datasources:
        resp.append(
            {
                "id": ds.id,
                "uid": ds.uid,
                "name": ds.name,
                "type": ds.type,
                "isDefault": ds.is_default,
            }
        )

    return resp


async def get_datasource_by_uid(uid: str) -> bytes:
    """
    Get a datasource by uid.
    """
    return await grafana_client.get_datasource(uid=uid)


async def get_datasource_by_name(name: str) -> bytes:
    """
    Get a datasource by name.
    """
    return await grafana_client.get_datasource(name=name)


def add_tools(mcp: FastMCP):
    mcp.add_tool(list_datasources)
    mcp.add_tool(get_datasource_by_uid)
    mcp.add_tool(get_datasource_by_name)
