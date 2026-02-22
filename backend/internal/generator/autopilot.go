package generator

// addAutopilotBoilerplate injects the "always needed" patterns every production
// backend repeats: request-id propagation, structured request logging, and a
// pagination helper. Called for both monolith and each microservice root.
func addAutopilotBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addGoAutopilot(tree, req, prefix)
	case "node":
		addNodeAutopilot(tree, req, prefix)
	case "python":
		addPythonAutopilot(tree, req, prefix)
	}
}

// ── Go ──────────────────────────────────────────────────────────────────────

func addGoAutopilot(tree *FileTree, req GenerateRequest, prefix string) {
	// X-Request-ID middleware
	if req.Framework == "fiber" {
		addFile(tree, prefix+"internal/middleware/requestid.go", `package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID attaches a unique X-Request-ID header to every request/response.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("X-Request-ID", id)
		c.Locals("requestID", id)
		return c.Next()
	}
}
`)
		addFile(tree, prefix+"internal/middleware/requestlogger.go", `package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger logs method, path, status code, and latency for every request.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		rid, _ := c.Locals("requestID").(string)
		fmt.Printf("[%s] %s %s → %d (%s) rid=%s\n",
			time.Now().Format(time.RFC3339),
			c.Method(), c.Path(),
			c.Response().StatusCode(),
			time.Since(start),
			rid,
		)
		return err
	}
}
`)
	} else {
		// Gin
		addFile(tree, prefix+"internal/middleware/requestid.go", `package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID attaches a unique X-Request-ID header to every request/response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Header("X-Request-ID", id)
		c.Set("requestID", id)
		c.Next()
	}
}
`)
		addFile(tree, prefix+"internal/middleware/requestlogger.go", `package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs method, path, status code, and latency for every request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		rid, _ := c.Get("requestID")
		fmt.Printf("[%s] %s %s → %d (%s) rid=%v\n",
			time.Now().Format(time.RFC3339),
			c.Request.Method, c.FullPath(),
			c.Writer.Status(),
			time.Since(start),
			rid,
		)
	}
}
`)
	}

	addFile(tree, prefix+"internal/pagination/pagination.go", `package pagination

// Page holds normalised pagination parameters parsed from query strings.
type Page struct {
	Limit  int
	Offset int
}

// Parse returns a Page with safe defaults: limit capped at 100, minimum 1.
func Parse(limit, offset int) Page {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}
`)
}

// ── Node ─────────────────────────────────────────────────────────────────────

func addNodeAutopilot(tree *FileTree, req GenerateRequest, prefix string) {
	_ = req
	addFile(tree, prefix+"src/middleware/requestId.js", `import { randomUUID } from 'node:crypto';

/**
 * Attaches a unique X-Request-ID header to every request/response.
 * Works with Express and Fastify (via onRequest hook).
 */
export function requestId(req, res, next) {
  const id = req.headers['x-request-id'] || randomUUID();
  req.requestId = id;
  res.setHeader('X-Request-ID', id);
  if (next) next();
}
`)
	addFile(tree, prefix+"src/middleware/requestLogger.js",
		"/**\n"+
			" * Logs method, path, status code, and latency for every request.\n"+
			" */\n"+
			"export function requestLogger(req, res, next) {\n"+
			"  const start = Date.now();\n"+
			"  res.on('finish', () => {\n"+
			"    const msg = '[' + new Date().toISOString() + '] ' + req.method + ' ' + req.url +\n"+
			"      ' -> ' + res.statusCode + ' (' + (Date.now() - start) + 'ms) rid=' + (req.requestId || '-');\n"+
			"    console.log(msg);\n"+
			"  });\n"+
			"  if (next) next();\n"+
			"}\n")
	addFile(tree, prefix+"src/utils/pagination.js", `/**
 * Parses limit/offset from a query object and returns safe defaults.
 * @param {{ limit?: string|number, offset?: string|number }} query
 * @returns {{ limit: number, offset: number }}
 */
export function parsePage(query = {}) {
  const limit = Math.min(Math.max(Number(query.limit) || 20, 1), 100);
  const offset = Math.max(Number(query.offset) || 0, 0);
  return { limit, offset };
}
`)
}

// ── Python ───────────────────────────────────────────────────────────────────

func addPythonAutopilot(tree *FileTree, req GenerateRequest, prefix string) {
	if req.Framework == "django" {
		// Django has its own middleware system
		addFile(tree, prefix+"api/middleware.py", `import uuid
from django.utils.deprecation import MiddlewareMixin
import time
import logging

logger = logging.getLogger(__name__)

class RequestIDMiddleware(MiddlewareMixin):
    """Attaches a unique X-Request-ID to every request and response."""
    def process_request(self, request):
        request_id = request.headers.get('X-Request-ID', str(uuid.uuid4()))
        request.request_id = request_id

    def process_response(self, request, response):
        rid = getattr(request, 'request_id', '-')
        response['X-Request-ID'] = rid
        return response

class RequestLoggerMiddleware(MiddlewareMixin):
    """Logs method, path, status, and latency for every request."""
    def process_request(self, request):
        request._start_time = time.monotonic()

    def process_response(self, request, response):
        duration_ms = (time.monotonic() - getattr(request, '_start_time', time.monotonic())) * 1000
        logger.info(
            "%s %s → %d (%.1fms) rid=%s",
            request.method, request.path, response.status_code,
            duration_ms, getattr(request, 'request_id', '-')
        )
        return response
`)
		addFile(tree, prefix+"api/pagination.py", `def parse_page(limit: int = 20, offset: int = 0) -> dict:
    """Returns normalised pagination params (limit capped at 100)."""
    return {"limit": min(max(limit, 1), 100), "offset": max(offset, 0)}
`)
		return
	}

	// FastAPI / Starlette
	addFile(tree, prefix+"app/middleware/request_id.py", `import uuid
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request

class RequestIDMiddleware(BaseHTTPMiddleware):
    """Attaches a unique X-Request-ID to every request and response."""
    async def dispatch(self, request: Request, call_next):
        request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        request.state.request_id = request_id
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        return response
`)
	addFile(tree, prefix+"app/middleware/request_logger.py", `import time
import logging
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request

logger = logging.getLogger("stacksprint")

class RequestLoggerMiddleware(BaseHTTPMiddleware):
    """Logs method, path, status code, and latency for every request."""
    async def dispatch(self, request: Request, call_next):
        start = time.monotonic()
        response = await call_next(request)
        duration_ms = (time.monotonic() - start) * 1000
        rid = getattr(request.state, "request_id", "-")
        logger.info(
            "%s %s → %d (%.1fms) rid=%s",
            request.method, request.url.path,
            response.status_code, duration_ms, rid
        )
        return response
`)
	addFile(tree, prefix+"app/utils/pagination.py", `from dataclasses import dataclass
from fastapi import Query

@dataclass
class PageParams:
    """Reusable pagination parameters for any list endpoint."""
    limit: int
    offset: int

def page_params(
    limit: int = Query(default=20, ge=1, le=100),
    offset: int = Query(default=0, ge=0),
) -> PageParams:
    """FastAPI dependency — inject with Depends(page_params)."""
    return PageParams(limit=limit, offset=offset)
`)
}

// addDBRetry generates a DB connection helper with exponential backoff for
// the given language. Injected whenever a database is selected.
func addDBRetry(tree *FileTree, req GenerateRequest, root string) {
	if req.Database == "none" {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/db/retry.go", `package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ConnectWithRetry retries sql.Open + Ping up to maxRetries times with
// exponential backoff. Use this in main() instead of a bare Connect().
func ConnectWithRetry(driver, dsn string, maxRetries int) (*sql.DB, error) {
	var db *sql.DB
	var err error
	wait := time.Second
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open(driver, dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				return db, nil
			}
		}
		fmt.Printf("DB not ready (attempt %d/%d): %v — retrying in %s\n", i+1, maxRetries, err, wait)
		time.Sleep(wait)
		wait *= 2
	}
	return nil, fmt.Errorf("database unavailable after %d retries: %w", maxRetries, err)
}
`)
	case "node":
		addFile(tree, prefix+"src/db/retry.js",
			"/**\n"+
				" * Retries a DB connect function with exponential back-off.\n"+
				" * @param {() => Promise<any>} connectFn\n"+
				" * @param {number} maxRetries\n"+
				" * @returns {Promise<any>}\n"+
				" */\n"+
				"export async function connectWithRetry(connectFn, maxRetries = 10) {\n"+
				"  let wait = 1000;\n"+
				"  for (let i = 0; i < maxRetries; i++) {\n"+
				"    try {\n"+
				"      return await connectFn();\n"+
				"    } catch (err) {\n"+
				"      console.log('DB not ready (attempt ' + (i+1) + '/' + maxRetries + '): ' + err.message + ' - retrying in ' + wait + 'ms');\n"+
				"      await new Promise((r) => setTimeout(r, wait));\n"+
				"      wait = Math.min(wait * 2, 16000);\n"+
				"    }\n"+
				"  }\n"+
				"  throw new Error('Database unavailable after ' + maxRetries + ' retries');\n"+
				"}\n")
	case "python":
		if req.Framework != "django" {
			addFile(tree, prefix+"app/db/retry.py", `import time
import logging

logger = logging.getLogger("stacksprint")

def connect_with_retry(connect_fn, max_retries: int = 10):
    """Calls connect_fn() with exponential back-off until it succeeds."""
    wait = 1.0
    for attempt in range(1, max_retries + 1):
        try:
            return connect_fn()
        except Exception as exc:
            logger.warning(
                "DB not ready (attempt %d/%d): %s — retrying in %.1fs",
                attempt, max_retries, exc, wait
            )
            time.sleep(wait)
            wait = min(wait * 2, 16)
    raise RuntimeError(f"Database unavailable after {max_retries} retries")
`)
		}
	}
}
