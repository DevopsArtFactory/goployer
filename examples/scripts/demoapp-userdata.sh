#!/bin/bash
set -euxo pipefail

dnf install -y python3

mkdir -p /opt/demoapp
cat >/opt/demoapp/app.py <<'PY'
#!/usr/bin/env python3
import json
import os
import socket
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

APP_VERSION = os.environ.get("APP_VERSION", "stable")


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/actuator/health":
            self.reply({"status": "UP"})
            return

        self.reply({
            "app": "demoapp",
            "version": APP_VERSION,
            "hostname": socket.gethostname(),
            "path": self.path,
        })

    def log_message(self, fmt, *args):
        return

    def reply(self, body):
        data = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
PY

cat >/etc/systemd/system/demoapp.service <<'UNIT'
[Unit]
Description=demoapp sample Python server
After=network-online.target
Wants=network-online.target

[Service]
Environment=APP_VERSION=stable
ExecStart=/usr/bin/python3 /opt/demoapp/app.py
Restart=always
RestartSec=2
User=root

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable --now demoapp
