#!/usr/bin/env python3

# NDFC Data Collector
# This script collects data from Cisco NDFC for health check analysis
# Generated from pkg/req/requests.go - do not edit manually

import json
import os
import re
import sys
import zipfile
from getpass import getpass

import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

BASE_URL = "/appcenter/cisco/ndfc/api/v1"

# Request definitions: (url_template, depends_on)
# url_template may contain {placeholder} names that are resolved from parent
# response items whose JSON field name matches the placeholder.
# Generated from pkg/req/requests.go - do not edit the list below manually.
REQUESTS = [
    ("/lan-fabric/rest/control/fabrics", None),
    ("/fm/about/version", None),
    ("/lan-fabric/rest/control/switches/overview", None),
    ("/lan-fabric/rest/control/fabrics/{fabricName}/inventory/switchesByFabric", "/lan-fabric/rest/control/fabrics"),
]

class NDFCClient:
    def __init__(self, host, username, password):
        self.host = host
        self.username = username
        self.password = password
        self.session = requests.Session()
        self.base_url = f"https://{host}"

    def login(self):
        login_url = f"{self.base_url}/login"
        payload = {
            "userName": self.username,
            "userPasswd": self.password,
            "domain": "DefaultAuth",
        }
        try:
            response = self.session.post(login_url, json=payload, verify=False)
            response.raise_for_status()
            print(f"Successfully authenticated to NDFC at {self.host}")
            return True
        except requests.exceptions.RequestException as e:
            print(f"Authentication failed: {e}")
            return False

    def get(self, endpoint):
        url = f"{self.base_url}{endpoint}"
        try:
            response = self.session.get(url, verify=False)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Request failed for {endpoint}: {e}")
            return None


def url_to_filename(url):
    """Convert URL to filename: /lan-fabric/rest/control/fabrics -> lan-fabric.rest.control.fabrics.json"""
    filename = url.strip('/').replace('/', '.')
    return f"{filename}.json"


def substitute_url(template, context):
    """Replace {placeholder} patterns in template using values from context."""
    def replace(match):
        key = match.group(1)
        return str(context.get(key, match.group(0)))
    return re.sub(r'\{([^}]+)\}', replace, template)


def item_to_context(item, parent_ctx=None):
    """Extract top-level scalar fields from a JSON object into a context dict."""
    ctx = dict(parent_ctx or {})
    if isinstance(item, dict):
        for key, value in item.items():
            if not isinstance(value, (dict, list)):
                ctx[key] = str(value)
    return ctx


def build_levels(request_defs):
    """Group requests into topological dependency levels for ordered execution."""
    by_url = {url: dep for url, dep in request_defs}
    depth = {}

    def calc_depth(url):
        if url in depth:
            return depth[url]
        dep = by_url.get(url)
        if dep is None:
            depth[url] = 0
        else:
            depth[url] = calc_depth(dep) + 1
        return depth[url]

    for url, _ in request_defs:
        calc_depth(url)

    max_depth = max(depth.values()) if depth else 0
    levels = [[] for _ in range(max_depth + 1)]
    for url, dep in request_defs:
        levels[depth[url]].append((url, dep))
    return levels


def collect_data(client):
    """Collect data from all endpoints, resolving dependent queries in order."""
    data = {}     # filename -> JSON content
    results = {}  # url_template -> list of (context, response) pairs

    levels = build_levels(REQUESTS)

    for level_reqs in levels:
        for url_template, depends_on in level_reqs:
            if depends_on is None:
                full_url = BASE_URL + url_template
                filename = url_to_filename(url_template)
                print(f"Fetching {filename}...")
                result = client.get(full_url)
                if result is not None:
                    data[filename] = result
                    results[url_template] = [({}, result)]
                    print(f"  \u2713 {filename} complete")
                else:
                    print(f"  \u2717 {filename} failed")
            else:
                parent_results = results.get(depends_on, [])
                level_results = []
                for parent_ctx, parent_data in parent_results:
                    items = parent_data if isinstance(parent_data, list) else [parent_data]
                    for item in items:
                        ctx = item_to_context(item, parent_ctx)
                        resolved_url = substitute_url(url_template, ctx)
                        full_url = BASE_URL + resolved_url
                        filename = url_to_filename(resolved_url)
                        print(f"Fetching {filename}...")
                        result = client.get(full_url)
                        if result is not None:
                            data[filename] = result
                            level_results.append((ctx, result))
                            print(f"  \u2713 {filename} complete")
                        else:
                            print(f"  \u2717 {filename} failed")
                if level_results:
                    results[url_template] = level_results

    return data


def main():
    print("NDFC Data Collector")
    print("=" * 50)
    print()

    # Get credentials
    host = input("NDFC hostname or IP: ").strip()
    username = input("NDFC username: ").strip()
    password = getpass("NDFC password: ")

    # Create client and login
    client = NDFCClient(host, username, password)
    if not client.login():
        sys.exit(1)

    print()
    print("Collecting data...")
    data = collect_data(client)

    # Create zip file
    output_file = "ndfc-collection-data.zip"
    print()
    print(f"Creating {output_file}...")

    with zipfile.ZipFile(output_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
        for filename, content in data.items():
            zipf.writestr(filename, json.dumps(content, indent=2))

    print()
    print("=" * 50)
    print("Collection complete!")
    print(f"Output written to {os.path.abspath(output_file)}")
    print("Please provide this file to Cisco Services for analysis.")


if __name__ == "__main__":
    main()
