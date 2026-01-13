# Node.js Plugin Development

This guide covers Node.js/JavaScript-specific best practices for Specular plugin development.

## Why Node.js?

- **Async First:** Native async/await for I/O operations
- **NPM Ecosystem:** Vast library availability
- **TypeScript:** Optional strong typing support
- **JSON Native:** Natural JSON handling

## Quick Start

```bash
specular plugin create my-plugin --type formatter --lang node
cd my-plugin
echo '{"action":"health"}' | node index.js
```

## Project Structure

```
my-plugin/
├── plugin.yaml       # Plugin manifest
├── package.json      # Node package config
├── index.js          # Entry point
├── src/
│   ├── handlers.js   # Action handlers
│   ├── config.js     # Configuration
│   └── utils.js      # Utilities
└── tests/
    └── handlers.test.js
```

### TypeScript Structure

```
my-plugin/
├── plugin.yaml
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts
│   ├── handlers.ts
│   ├── config.ts
│   ├── types.ts
│   └── utils.ts
├── dist/            # Compiled output
└── tests/
    └── handlers.test.ts
```

## Code Templates

### Basic Plugin (JavaScript)

```javascript
#!/usr/bin/env node
/**
 * My Plugin - A Specular plugin
 */

const readline = require('readline');

const VERSION = '1.0.0';

/**
 * Handle incoming request
 * @param {Object} request - Plugin request
 * @returns {Object} Plugin response
 */
async function handleRequest(request) {
  const action = request.action || '';

  const handlers = {
    health: handleHealth,
    format: handleFormat,
  };

  const handler = handlers[action];
  if (!handler) {
    return {
      success: false,
      error: `unknown action: ${action}`,
    };
  }

  try {
    const result = await handler(request);
    return { success: true, result };
  } catch (error) {
    return {
      success: false,
      error: error.message,
    };
  }
}

/**
 * Handle health check
 */
function handleHealth() {
  return {
    status: 'healthy',
    version: VERSION,
  };
}

/**
 * Handle format request
 */
async function handleFormat(request) {
  const { data = {}, config = {} } = request;
  const { content, metadata } = data;

  // Your formatting logic here
  const output = formatContent(content, config);

  return {
    output,
    format: config.format || 'text',
  };
}

function formatContent(content, config) {
  // Implementation
  return content;
}

/**
 * Main entry point
 */
async function main() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false,
  });

  for await (const line of rl) {
    let response;

    try {
      const request = JSON.parse(line);
      response = await handleRequest(request);
    } catch (error) {
      response = {
        success: false,
        error: `json: ${error.message}`,
      };
    }

    console.log(JSON.stringify(response));
  }
}

main().catch((error) => {
  console.error(`Fatal error: ${error.message}`);
  process.exit(1);
});
```

### TypeScript Plugin

```typescript
#!/usr/bin/env node
/**
 * My Plugin - A Specular plugin (TypeScript)
 */

import * as readline from 'readline';

const VERSION = '1.0.0';

// Types
interface PluginRequest {
  action: string;
  params?: Record<string, unknown>;
  data?: Record<string, unknown>;
  config?: Record<string, unknown>;
}

interface PluginResponse {
  success: boolean;
  result?: unknown;
  error?: string;
}

interface HealthResult {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
}

// Handlers
type Handler = (request: PluginRequest) => Promise<unknown>;

const handlers: Record<string, Handler> = {
  health: handleHealth,
  format: handleFormat,
};

async function handleHealth(): Promise<HealthResult> {
  return {
    status: 'healthy',
    version: VERSION,
  };
}

async function handleFormat(request: PluginRequest): Promise<{ output: string }> {
  const content = request.data?.content as string || '';
  const format = request.config?.format as string || 'text';

  // Your formatting logic
  const output = formatContent(content, format);

  return { output };
}

function formatContent(content: string, format: string): string {
  // Implementation
  return content;
}

// Main
async function handleRequest(request: PluginRequest): Promise<PluginResponse> {
  const handler = handlers[request.action];

  if (!handler) {
    return {
      success: false,
      error: `unknown action: ${request.action}`,
    };
  }

  try {
    const result = await handler(request);
    return { success: true, result };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

async function main(): Promise<void> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false,
  });

  for await (const line of rl) {
    let response: PluginResponse;

    try {
      const request: PluginRequest = JSON.parse(line);
      response = await handleRequest(request);
    } catch (error) {
      response = {
        success: false,
        error: `json: ${error instanceof Error ? error.message : String(error)}`,
      };
    }

    console.log(JSON.stringify(response));
  }
}

main().catch((error) => {
  console.error(`Fatal: ${error}`);
  process.exit(1);
});
```

### Custom Errors

```typescript
// errors.ts

export class PluginError extends Error {
  category: string;

  constructor(category: string, message: string) {
    super(`${category}: ${message}`);
    this.category = category;
    this.name = 'PluginError';
  }
}

export class ConfigError extends PluginError {
  constructor(message: string) {
    super('config', message);
  }
}

export class ValidationError extends PluginError {
  constructor(message: string) {
    super('validation', message);
  }
}

export class NetworkError extends PluginError {
  constructor(message: string) {
    super('network', message);
  }
}
```

### Configuration Handling

```typescript
// config.ts

import { ConfigError } from './errors';

export interface Config {
  apiKey: string;
  endpoint: string;
  timeout: number;
  retries: number;
}

const defaultConfig: Omit<Config, 'apiKey'> = {
  endpoint: 'https://api.example.com',
  timeout: 30000,
  retries: 3,
};

export function parseConfig(raw: Record<string, unknown>): Config {
  const apiKey = raw.api_key;
  if (typeof apiKey !== 'string' || !apiKey) {
    throw new ConfigError('api_key is required');
  }

  const config: Config = {
    apiKey,
    endpoint: typeof raw.endpoint === 'string'
      ? raw.endpoint
      : defaultConfig.endpoint,
    timeout: typeof raw.timeout === 'number'
      ? raw.timeout * 1000  // Convert to ms
      : defaultConfig.timeout,
    retries: typeof raw.retries === 'number'
      ? raw.retries
      : defaultConfig.retries,
  };

  validateConfig(config);
  return config;
}

function validateConfig(config: Config): void {
  if (config.timeout <= 0) {
    throw new ConfigError('timeout must be positive');
  }
  if (config.retries < 0) {
    throw new ConfigError('retries cannot be negative');
  }
}
```

### HTTP Client with Retry

```typescript
// client.ts

import { Config } from './config';
import { NetworkError } from './errors';

interface RequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  path: string;
  body?: unknown;
}

export class APIClient {
  private config: Config;

  constructor(config: Config) {
    this.config = config;
  }

  async request<T>(options: RequestOptions): Promise<T> {
    const url = `${this.config.endpoint}${options.path}`;

    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= this.config.retries; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(
          () => controller.abort(),
          this.config.timeout
        );

        const response = await fetch(url, {
          method: options.method,
          headers: {
            'Authorization': `Bearer ${this.config.apiKey}`,
            'Content-Type': 'application/json',
          },
          body: options.body ? JSON.stringify(options.body) : undefined,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        return await response.json() as T;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));

        if (attempt < this.config.retries) {
          await this.delay(Math.pow(2, attempt) * 100);
        }
      }
    }

    throw new NetworkError(lastError?.message || 'Request failed');
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

### Async Input Handling

```typescript
// Using async iterators for stdin
import { createInterface } from 'readline';

async function* readLines(): AsyncGenerator<string> {
  const rl = createInterface({
    input: process.stdin,
    crlfDelay: Infinity,
  });

  for await (const line of rl) {
    yield line;
  }
}

async function main(): Promise<void> {
  for await (const line of readLines()) {
    // Process each line
    const request = JSON.parse(line);
    const response = await handleRequest(request);
    console.log(JSON.stringify(response));
  }
}
```

### Logging

```typescript
// logger.ts

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

class Logger {
  private debugEnabled: boolean;

  constructor() {
    this.debugEnabled = process.env.DEBUG === 'true';
  }

  debug(message: string, ...args: unknown[]): void {
    if (this.debugEnabled) {
      this.log('debug', message, ...args);
    }
  }

  info(message: string, ...args: unknown[]): void {
    this.log('info', message, ...args);
  }

  warn(message: string, ...args: unknown[]): void {
    this.log('warn', message, ...args);
  }

  error(message: string, ...args: unknown[]): void {
    this.log('error', message, ...args);
  }

  private log(level: LogLevel, message: string, ...args: unknown[]): void {
    // Always write to stderr, never stdout
    const prefix = `[${level.toUpperCase()}]`;
    console.error(prefix, message, ...args);
  }
}

export const logger = new Logger();
```

## Testing

### Jest Configuration

```json
// package.json
{
  "scripts": {
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  },
  "jest": {
    "testEnvironment": "node",
    "transform": {
      "^.+\\.tsx?$": "ts-jest"
    },
    "testMatch": ["**/tests/**/*.test.ts"]
  }
}
```

### Unit Tests

```typescript
// tests/handlers.test.ts

import { handleRequest } from '../src/index';

describe('Plugin Handlers', () => {
  describe('health', () => {
    it('returns healthy status', async () => {
      const response = await handleRequest({ action: 'health' });

      expect(response.success).toBe(true);
      expect(response.result).toEqual({
        status: 'healthy',
        version: expect.any(String),
      });
    });
  });

  describe('format', () => {
    it('formats content successfully', async () => {
      const request = {
        action: 'format',
        data: { content: 'test content' },
        config: { format: 'html' },
      };

      const response = await handleRequest(request);

      expect(response.success).toBe(true);
      expect(response.result).toHaveProperty('output');
    });

    it('handles missing content', async () => {
      const request = {
        action: 'format',
        data: {},
      };

      const response = await handleRequest(request);

      expect(response.success).toBe(false);
    });
  });

  describe('unknown action', () => {
    it('returns error for unknown action', async () => {
      const response = await handleRequest({ action: 'invalid' });

      expect(response.success).toBe(false);
      expect(response.error).toContain('unknown action');
    });
  });
});
```

### Integration Tests

```typescript
// tests/integration.test.ts

import { spawn, ChildProcess } from 'child_process';
import * as readline from 'readline';

describe('Plugin Integration', () => {
  let proc: ChildProcess;
  let rl: readline.Interface;

  beforeAll(() => {
    proc = spawn('node', ['dist/index.js'], {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    rl = readline.createInterface({
      input: proc.stdout!,
    });
  });

  afterAll(() => {
    proc.kill();
  });

  async function sendRequest(request: object): Promise<object> {
    return new Promise((resolve) => {
      rl.once('line', (line) => {
        resolve(JSON.parse(line));
      });
      proc.stdin!.write(JSON.stringify(request) + '\n');
    });
  }

  it('responds to health check', async () => {
    const response = await sendRequest({ action: 'health' });
    expect(response).toHaveProperty('success', true);
  });

  it('processes format request', async () => {
    const response = await sendRequest({
      action: 'format',
      data: { content: 'test' },
    });
    expect(response).toHaveProperty('success', true);
  });
});
```

### Mocking with Jest

```typescript
// tests/client.test.ts

import { APIClient } from '../src/client';

// Mock fetch globally
global.fetch = jest.fn();

describe('APIClient', () => {
  const config = {
    apiKey: 'test-key',
    endpoint: 'https://api.test.com',
    timeout: 5000,
    retries: 2,
  };

  let client: APIClient;

  beforeEach(() => {
    client = new APIClient(config);
    jest.clearAllMocks();
  });

  it('makes successful request', async () => {
    (fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: 'test' }),
    });

    const result = await client.request({
      method: 'GET',
      path: '/test',
    });

    expect(result).toEqual({ data: 'test' });
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it('retries on failure', async () => {
    (fetch as jest.Mock)
      .mockRejectedValueOnce(new Error('Network error'))
      .mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: 'success' }),
      });

    const result = await client.request({
      method: 'GET',
      path: '/test',
    });

    expect(result).toEqual({ data: 'success' });
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
```

## Package Configuration

### package.json

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "A Specular plugin",
  "main": "dist/index.js",
  "bin": {
    "my-plugin": "dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js",
    "dev": "ts-node src/index.ts",
    "test": "jest",
    "lint": "eslint src/",
    "typecheck": "tsc --noEmit"
  },
  "engines": {
    "node": ">=18.0.0"
  },
  "dependencies": {},
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "jest": "^29.0.0",
    "ts-jest": "^29.0.0",
    "@types/jest": "^29.0.0",
    "eslint": "^8.0.0"
  }
}
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "tests"]
}
```

## Performance Tips

1. **Use native fetch** (Node 18+) instead of axios/node-fetch
2. **Stream large responses** instead of buffering
3. **Use worker threads** for CPU-intensive operations
4. **Minimize dependencies** to reduce startup time

```javascript
// Streaming example
async function* streamResponse(response) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    yield decoder.decode(value);
  }
}
```

## Common Pitfalls

1. **Not awaiting async operations** - Always await or handle promises
2. **console.log for debugging** - Use console.error for logs
3. **Unhandled promise rejections** - Add global error handler
4. **Blocking the event loop** - Use setImmediate for CPU work
5. **Memory leaks** - Close streams and clear intervals
