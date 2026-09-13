import asyncio
import json
import unittest
from typing import Any

from fastapi import FastAPI

from app.interfaces.errors.exception_handler import register_exception_handlers
from app.interfaces.errors.exceptions import AppException
from app.interfaces.service_dependencies import (
    get_file_service,
    get_shell_service,
    get_supervisor_service,
)
from app.main import app
from app.models.file import FileReadResult, FileWriteResult, FileDeleteResult
from app.models.shell import ShellExecuteResult
from app.models.supervisor import ProcessInfo, SupervisorTimeout


class FakeFileService:
    async def read_file(self, **kwargs):
        return FileReadResult(filepath=kwargs["filepath"], content="hello")

    async def write_file(self, **kwargs):
        return FileWriteResult(filepath=kwargs["filepath"], bytes_written=len(kwargs["content"]))

    async def delete_file(self, **kwargs):
        return FileDeleteResult(filepath=kwargs["filepath"], deleted=True)

    async def ensure_file(self, filepath):
        return None


class FakeShellService:
    def create_session_id(self):
        return "generated-session"

    async def exec_command(self, **kwargs):
        return ShellExecuteResult(
            session_id=kwargs["session_id"], command=kwargs["command"], status="running"
        )


class FakeSupervisorService:
    async def get_timeout_status(self):
        return SupervisorTimeout(active=False)


class ApiEndpointTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.file_service = FakeFileService()
        cls.shell_service = FakeShellService()
        cls.supervisor_service = FakeSupervisorService()
        app.dependency_overrides[get_file_service] = lambda: cls.file_service
        app.dependency_overrides[get_shell_service] = lambda: cls.shell_service
        app.dependency_overrides[get_supervisor_service] = lambda: cls.supervisor_service

    @classmethod
    def tearDownClass(cls):
        app.dependency_overrides.clear()

    async def request(self, method: str, path: str, payload: Any = None):
        """通过 ASGI 协议直接调用应用，避免测试依赖额外 HTTP 客户端。"""
        body = b"" if payload is None else json.dumps(payload).encode("utf-8")
        messages = [{"type": "http.request", "body": body, "more_body": False}]
        response = {"status": None, "body": bytearray()}

        async def receive():
            if messages:
                return messages.pop(0)
            await asyncio.sleep(0)
            return {"type": "http.disconnect"}

        async def send(message):
            if message["type"] == "http.response.start":
                response["status"] = message["status"]
            elif message["type"] == "http.response.body":
                response["body"].extend(message.get("body", b""))

        headers = [(b"content-type", b"application/json")] if body else []
        scope = {
            "type": "http", "asgi": {"version": "3.0"}, "http_version": "1.1",
            "method": method, "scheme": "http", "path": path, "raw_path": path.encode(),
            "query_string": b"", "headers": headers, "client": ("test", 1),
            "server": ("test", 80),
        }
        await app(scope, receive, send)
        content = bytes(response["body"])
        return response["status"], json.loads(content) if content else None

    def run_request(self, method: str, path: str, payload: Any = None):
        return asyncio.run(self.request(method, path, payload))

    def test_health_endpoint_returns_status(self):
        status, body = self.run_request("GET", "/health")
        self.assertEqual(status, 200)
        self.assertEqual(body, {"status": "healthy"})

    def test_file_read_endpoint_returns_envelope(self):
        status, body = self.run_request("POST", "/api/file/read-file", {"filepath": "/tmp/demo.txt"})
        self.assertEqual(status, 200)
        self.assertEqual(body["code"], 200)
        self.assertEqual(body["data"]["content"], "hello")

    def test_file_read_endpoint_rejects_invalid_max_length(self):
        status, body = self.run_request(
            "POST", "/api/file/read-file", {"filepath": "/tmp/demo.txt", "max_length": 0}
        )
        self.assertEqual(status, 422)
        self.assertEqual(body["code"], 422)

    def test_file_write_and_delete_endpoints_return_results(self):
        write_status, write_body = self.run_request(
            "POST", "/api/file/write-file", {"filepath": "/tmp/demo.txt", "content": "hello"}
        )
        delete_status, delete_body = self.run_request(
            "DELETE", "/api/file/delete-file", {"filepath": "/tmp/demo.txt"}
        )
        self.assertEqual(write_status, 200)
        self.assertEqual(write_body["data"]["bytes_written"], 5)
        self.assertEqual(delete_status, 200)
        self.assertTrue(delete_body["data"]["deleted"])

    def test_shell_exec_endpoint_generates_session_id(self):
        status, body = self.run_request("POST", "/api/shell/exec-command", {"command": "printf hello"})
        self.assertEqual(status, 200)
        self.assertEqual(body["data"]["session_id"], "generated-session")

    def test_supervisor_timeout_status_endpoint_returns_envelope(self):
        status, body = self.run_request("GET", "/api/supervisor/timeout-status")
        self.assertEqual(status, 200)
        self.assertFalse(body["data"]["active"])

    def test_app_exception_handler_preserves_error_data(self):
        test_app = FastAPI()
        register_exception_handlers(test_app)
        handler = test_app.exception_handlers[AppException]
        request = type("Request", (), {})()

        response = asyncio.run(handler(request, AppException("failed", data={"request_id": "r1"})))

        self.assertEqual(json.loads(response.body)["data"], {"request_id": "r1"})


if __name__ == "__main__":
    unittest.main()
