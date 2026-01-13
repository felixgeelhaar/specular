#!/usr/bin/env node
/**
 * Markdown Formatter - A Specular formatter plugin
 *
 * This plugin formats content as Markdown with customizable options:
 * - Table of contents generation
 * - Metadata headers
 * - Configurable heading levels
 */

const readline = require('readline');

const VERSION = '1.0.0';

/**
 * Default configuration
 */
const DEFAULT_CONFIG = {
  include_toc: false,
  include_metadata: true,
  heading_level: 1,
  code_fence_style: 'backtick',
};

/**
 * Parse configuration from request
 * @param {Object} raw - Raw configuration object
 * @returns {Object} Parsed configuration
 */
function parseConfig(raw) {
  return {
    includeToc: raw.include_toc ?? DEFAULT_CONFIG.include_toc,
    includeMetadata: raw.include_metadata ?? DEFAULT_CONFIG.include_metadata,
    headingLevel: Math.min(6, Math.max(1, raw.heading_level ?? DEFAULT_CONFIG.heading_level)),
    codeFenceStyle: raw.code_fence_style ?? DEFAULT_CONFIG.code_fence_style,
  };
}

/**
 * Generate heading with specified level
 * @param {string} text - Heading text
 * @param {number} level - Heading level (1-6)
 * @returns {string} Markdown heading
 */
function heading(text, level) {
  const prefix = '#'.repeat(Math.min(6, Math.max(1, level)));
  return `${prefix} ${text}`;
}

/**
 * Generate code fence
 * @param {string} code - Code content
 * @param {string} language - Language identifier
 * @param {string} style - Fence style (backtick or tilde)
 * @returns {string} Fenced code block
 */
function codeBlock(code, language, style) {
  const fence = style === 'tilde' ? '~~~' : '```';
  return `${fence}${language}\n${code}\n${fence}`;
}

/**
 * Generate table of contents from headings
 * @param {string} content - Markdown content
 * @returns {string} Table of contents
 */
function generateToc(content) {
  const headingRegex = /^(#{1,6})\s+(.+)$/gm;
  const toc = [];
  let match;

  while ((match = headingRegex.exec(content)) !== null) {
    const level = match[1].length;
    const text = match[2];
    const anchor = text.toLowerCase().replace(/[^a-z0-9]+/g, '-');
    const indent = '  '.repeat(level - 1);
    toc.push(`${indent}- [${text}](#${anchor})`);
  }

  if (toc.length === 0) {
    return '';
  }

  return '## Table of Contents\n\n' + toc.join('\n') + '\n';
}

/**
 * Generate metadata header
 * @param {Object} metadata - Metadata object
 * @returns {string} YAML front matter
 */
function generateMetadata(metadata) {
  if (!metadata || Object.keys(metadata).length === 0) {
    return '';
  }

  const lines = ['---'];
  for (const [key, value] of Object.entries(metadata)) {
    if (value !== undefined && value !== null) {
      lines.push(`${key}: ${JSON.stringify(value)}`);
    }
  }
  lines.push('---', '');

  return lines.join('\n');
}

/**
 * Format content as Markdown
 * @param {Object} data - Input data
 * @param {Object} config - Configuration
 * @returns {string} Formatted Markdown
 */
function formatMarkdown(data, config) {
  const parts = [];

  // Add metadata if configured
  if (config.includeMetadata && data.metadata) {
    const meta = generateMetadata(data.metadata);
    if (meta) {
      parts.push(meta);
    }
  }

  // Add title
  if (data.title) {
    parts.push(heading(data.title, config.headingLevel));
    parts.push('');
  }

  // Add description
  if (data.description) {
    parts.push(data.description);
    parts.push('');
  }

  // Add main content
  if (data.content) {
    // Handle different content types
    if (typeof data.content === 'string') {
      parts.push(data.content);
    } else if (Array.isArray(data.content)) {
      // Handle array of sections
      for (const section of data.content) {
        if (section.heading) {
          parts.push(heading(section.heading, config.headingLevel + 1));
          parts.push('');
        }
        if (section.text) {
          parts.push(section.text);
          parts.push('');
        }
        if (section.code) {
          parts.push(codeBlock(section.code, section.language || '', config.codeFenceStyle));
          parts.push('');
        }
      }
    } else if (typeof data.content === 'object') {
      // Handle object content
      parts.push(codeBlock(JSON.stringify(data.content, null, 2), 'json', config.codeFenceStyle));
    }
    parts.push('');
  }

  let markdown = parts.join('\n').trim() + '\n';

  // Add table of contents if configured
  if (config.includeToc) {
    const toc = generateToc(markdown);
    if (toc) {
      // Insert TOC after metadata and title
      const titleMatch = markdown.match(/^(---[\s\S]*?---\n)?(#.*\n\n)?/);
      if (titleMatch) {
        const insertPos = titleMatch[0].length;
        markdown = markdown.slice(0, insertPos) + toc + '\n' + markdown.slice(insertPos);
      } else {
        markdown = toc + '\n' + markdown;
      }
    }
  }

  return markdown;
}

/**
 * Handle health check
 * @returns {Object} Health status
 */
function handleHealth() {
  return {
    status: 'healthy',
    version: VERSION,
    name: 'markdown-formatter',
  };
}

/**
 * Handle format request
 * @param {Object} request - Plugin request
 * @returns {Object} Format result
 */
function handleFormat(request) {
  const data = request.data || {};
  const rawConfig = request.config || {};

  if (!data.content && !data.title) {
    throw new Error('validation: data must contain content or title');
  }

  const config = parseConfig(rawConfig);
  const output = formatMarkdown(data, config);

  return {
    output,
    format: 'markdown',
    metadata: {
      length: output.length,
      lines: output.split('\n').length,
    },
  };
}

/**
 * Handle incoming request
 * @param {Object} request - Plugin request
 * @returns {Object} Response
 */
async function handleRequest(request) {
  const action = request.action || '';

  switch (action) {
    case 'health':
      return { success: true, result: handleHealth() };

    case 'format':
      try {
        const result = handleFormat(request);
        return { success: true, result };
      } catch (error) {
        return { success: false, error: error.message };
      }

    default:
      return { success: false, error: `unknown action: ${action}` };
  }
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
  console.error(`Fatal: ${error.message}`);
  process.exit(1);
});
