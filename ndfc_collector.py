#!/usr/bin/env python3

# NDFC Data Collector
# This script collects data from Cisco NDFC for health check analysis

import json
import os
import sys
import zipfile
from getpass import getpass

import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


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

    # Define endpoints to query
    endpoints = [
        "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics",
        "/appcenter/cisco/ndfc/api/v1/fm/about/version",
        "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/switches/overview",
    ]

    # Collect data from each endpoint
    data = {}
    for endpoint in endpoints:
        filename = url_to_filename(endpoint.replace('/appcenter/cisco/ndfc/api/v1', ''))
        print(f"Fetching {filename}...")
        result = client.get(endpoint)
        if result is not None:
            data[filename] = result
            print(f"  ✓ {filename} complete")
        else:
            print(f"  ✗ {filename} failed")

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
