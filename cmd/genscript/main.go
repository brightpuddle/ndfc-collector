// Command genscript generates the ndfc_collector.py Python script
// from the embedded request data.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"ndfc-collector/pkg/req"
)

func main() {
	// Determine output path - should be at repo root
	// When run via go generate from pkg/req, we need to go up two directories
	scriptPath := "ndfc_collector.py"
	if _, err := os.Stat("../../go.mod"); err == nil {
		// We're in a subdirectory (e.g., pkg/req), write to repo root
		scriptPath = "../../ndfc_collector.py"
	}

	// Get absolute path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Open output file
	f, err := os.Create(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Write script header
	fmt.Fprintln(f, "#!/usr/bin/env python3")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "# NDFC Data Collector")
	fmt.Fprintln(f, "# This script collects data from Cisco NDFC for health check analysis")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "import json")
	fmt.Fprintln(f, "import os")
	fmt.Fprintln(f, "import sys")
	fmt.Fprintln(f, "import zipfile")
	fmt.Fprintln(f, "from getpass import getpass")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "import requests")
	fmt.Fprintln(f, "import urllib3")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "class NDFCClient:")
	fmt.Fprintln(f, "    def __init__(self, host, username, password):")
	fmt.Fprintln(f, "        self.host = host")
	fmt.Fprintln(f, "        self.username = username")
	fmt.Fprintln(f, "        self.password = password")
	fmt.Fprintln(f, "        self.session = requests.Session()")
	fmt.Fprintln(f, "        self.base_url = f\"https://{host}\"")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    def login(self):")
	fmt.Fprintln(f, "        login_url = f\"{self.base_url}/login\"")
	fmt.Fprintln(f, "        payload = {")
	fmt.Fprintln(f, "            \"userName\": self.username,")
	fmt.Fprintln(f, "            \"userPasswd\": self.password,")
	fmt.Fprintln(f, "            \"domain\": \"DefaultAuth\",")
	fmt.Fprintln(f, "        }")
	fmt.Fprintln(f, "        try:")
	fmt.Fprintln(f, "            response = self.session.post(login_url, json=payload, verify=False)")
	fmt.Fprintln(f, "            response.raise_for_status()")
	fmt.Fprintln(f, "            print(f\"Successfully authenticated to NDFC at {self.host}\")")
	fmt.Fprintln(f, "            return True")
	fmt.Fprintln(f, "        except requests.exceptions.RequestException as e:")
	fmt.Fprintln(f, "            print(f\"Authentication failed: {e}\")")
	fmt.Fprintln(f, "            return False")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    def get(self, endpoint):")
	fmt.Fprintln(f, "        url = f\"{self.base_url}{endpoint}\"")
	fmt.Fprintln(f, "        try:")
	fmt.Fprintln(f, "            response = self.session.get(url, verify=False)")
	fmt.Fprintln(f, "            response.raise_for_status()")
	fmt.Fprintln(f, "            return response.json()")
	fmt.Fprintln(f, "        except requests.exceptions.RequestException as e:")
	fmt.Fprintln(f, "            print(f\"Request failed for {endpoint}: {e}\")")
	fmt.Fprintln(f, "            return None")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "def url_to_filename(url):")
	fmt.Fprintln(f, "    \"\"\"Convert URL to filename: /lan-fabric/rest/control/fabrics -> lan-fabric.rest.control.fabrics.json\"\"\"")
	fmt.Fprintln(f, "    filename = url.strip('/').replace('/', '.')")
	fmt.Fprintln(f, "    return f\"{filename}.json\"")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "def main():")
	fmt.Fprintln(f, "    print(\"NDFC Data Collector\")")
	fmt.Fprintln(f, "    print(\"=\" * 50)")
	fmt.Fprintln(f, "    print()")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    # Get credentials")
	fmt.Fprintln(f, "    host = input(\"NDFC hostname or IP: \").strip()")
	fmt.Fprintln(f, "    username = input(\"NDFC username: \").strip()")
	fmt.Fprintln(f, "    password = getpass(\"NDFC password: \")")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    # Create client and login")
	fmt.Fprintln(f, "    client = NDFCClient(host, username, password)")
	fmt.Fprintln(f, "    if not client.login():")
	fmt.Fprintln(f, "        sys.exit(1)")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    print()")
	fmt.Fprintln(f, "    print(\"Collecting data...\")")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    # Define endpoints to query")
	fmt.Fprintln(f, "    endpoints = [")

	// Get requests
	reqs, err := req.GetRequests()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting requests: %v\n", err)
		os.Exit(1)
	}

	// Write endpoints
	for _, r := range reqs {
		fmt.Fprintf(f, "        \"%s%s\",\n", req.BaseURL, r.URL)
	}

	fmt.Fprintln(f, "    ]")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    # Collect data from each endpoint")
	fmt.Fprintln(f, "    data = {}")
	fmt.Fprintln(f, "    for endpoint in endpoints:")
	fmt.Fprintln(f, "        filename = url_to_filename(endpoint.replace('/appcenter/cisco/ndfc/api/v1', ''))")
	fmt.Fprintln(f, "        print(f\"Fetching {filename}...\")")
	fmt.Fprintln(f, "        result = client.get(endpoint)")
	fmt.Fprintln(f, "        if result is not None:")
	fmt.Fprintln(f, "            data[filename] = result")
	fmt.Fprintln(f, "            print(f\"  ✓ {filename} complete\")")
	fmt.Fprintln(f, "        else:")
	fmt.Fprintln(f, "            print(f\"  ✗ {filename} failed\")")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    # Create zip file")
	fmt.Fprintln(f, "    output_file = \"ndfc-collection-data.zip\"")
	fmt.Fprintln(f, "    print()")
	fmt.Fprintln(f, "    print(f\"Creating {output_file}...\")")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    with zipfile.ZipFile(output_file, 'w', zipfile.ZIP_DEFLATED) as zipf:")
	fmt.Fprintln(f, "        for filename, content in data.items():")
	fmt.Fprintln(f, "            zipf.writestr(filename, json.dumps(content, indent=2))")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "    print()")
	fmt.Fprintln(f, "    print(\"=\" * 50)")
	fmt.Fprintln(f, "    print(\"Collection complete!\")")
	fmt.Fprintln(f, "    print(f\"Output written to {os.path.abspath(output_file)}\")")
	fmt.Fprintln(f, "    print(\"Please provide this file to Cisco Services for analysis.\")")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "if __name__ == \"__main__\":")
	fmt.Fprintln(f, "    main()")

	// Make the script executable
	if err := os.Chmod(absPath, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error making script executable: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s successfully\n", absPath)
}
