// Specular CLI Command Reference Data

export interface Flag {
  name: string;
  shorthand?: string;
  description: string;
  default?: string;
}

export interface Command {
  name: string;
  description: string;
  usage: string;
  examples: string[];
  flags: Flag[];
  subcommands?: Command[];
}

export interface CommandGroup {
  id: string;
  name: string;
  icon: string;
  description: string;
  commands: Command[];
}

export const globalFlags: Flag[] = [
  { name: "--format", description: "Output format (text, json, yaml)", default: "text" },
  { name: "--verbose", shorthand: "-v", description: "Enable verbose output" },
  { name: "--quiet", shorthand: "-q", description: "Suppress non-essential output" },
  { name: "--explain", description: "Show AI reasoning and decision-making process" },
  { name: "--home", description: "Override .specular directory location" },
  { name: "--log-level", description: "Log level (debug, info, warn, error)", default: "info" },
  { name: "--no-color", description: "Disable colored output" },
  { name: "--trace", description: "Distributed tracing ID for debugging" },
];

export const commandGroups: CommandGroup[] = [
  {
    id: "getting-started",
    name: "Getting Started",
    icon: "rocket",
    description: "Initialize and configure your Specular project",
    commands: [
      {
        name: "init",
        description: "Initialize a new Specular project with smart context detection. Automatically detects your environment (Docker, AI providers, languages, frameworks, Git, CI) and generates optimized configuration files.",
        usage: "specular init [directory] [flags]",
        examples: [
          "specular init",
          "specular init --local",
          "specular init --cloud",
          "specular init --template web-app",
          "specular init --providers ollama,anthropic",
          "specular init --dry-run",
        ],
        flags: [
          { name: "--local", description: "Prefer local AI providers (Ollama)" },
          { name: "--cloud", description: "Prefer cloud AI providers (OpenAI, Anthropic, Gemini)" },
          { name: "--template", description: "Project template (web-app, api-service, cli-tool, microservice, data-pipeline)" },
          { name: "--providers", description: "Comma-separated list of providers to enable" },
          { name: "--governance", description: "Target governance level (L2, L3, L4)", default: "L2" },
          { name: "--dry-run", description: "Preview changes without writing files" },
          { name: "--no-detect", description: "Skip automatic context detection" },
          { name: "--yes", description: "Auto-accept all prompts (non-interactive mode)" },
          { name: "--force", shorthand: "-f", description: "Overwrite existing configuration files" },
          { name: "--mcp", description: "MCP integration (enable, disable, auto)", default: "auto" },
        ],
      },
      {
        name: "config",
        description: "Manage Specular global configuration stored at ~/.specular/config.yaml. Includes default provider preferences, budget limits, output format, and API credentials.",
        usage: "specular config [command]",
        examples: [
          "specular config view",
          "specular config edit",
          "specular config get default_provider",
          "specular config set default_provider ollama",
          "specular config path",
        ],
        flags: [],
        subcommands: [
          { name: "view", description: "Display current configuration", usage: "specular config view", examples: [], flags: [] },
          { name: "edit", description: "Edit configuration in $EDITOR", usage: "specular config edit", examples: [], flags: [] },
          { name: "get", description: "Get a specific configuration value", usage: "specular config get <key>", examples: [], flags: [] },
          { name: "set", description: "Set a specific configuration value", usage: "specular config set <key> <value>", examples: [], flags: [] },
          { name: "path", description: "Show configuration file path", usage: "specular config path", examples: [], flags: [] },
        ],
      },
    ],
  },
  {
    id: "specification",
    name: "Specification",
    icon: "document",
    description: "Generate, validate, and manage product specifications",
    commands: [
      {
        name: "spec new",
        description: "Create a new specification file with guided prompts or from a template.",
        usage: "specular spec new [flags]",
        examples: [
          "specular spec new",
          "specular spec new --template api-service",
        ],
        flags: [
          { name: "--template", description: "Use a specification template" },
        ],
      },
      {
        name: "spec generate",
        description: "Generate a structured specification from a PRD markdown file using AI.",
        usage: "specular spec generate <prd-file> [flags]",
        examples: [
          "specular spec generate requirements.md",
          "specular spec generate prd.md --output spec.yaml",
        ],
        flags: [
          { name: "--output", shorthand: "-o", description: "Output specification file path" },
        ],
      },
      {
        name: "spec validate",
        description: "Validate a specification file for correctness and completeness.",
        usage: "specular spec validate [file] [flags]",
        examples: [
          "specular spec validate",
          "specular spec validate .specular/spec.yaml",
        ],
        flags: [
          { name: "--strict", description: "Enable strict validation mode" },
        ],
      },
      {
        name: "spec approve",
        description: "Approve the current specification for use in planning and execution.",
        usage: "specular spec approve [flags]",
        examples: [
          "specular spec approve",
          "specular spec approve --message 'Ready for v1.0'",
        ],
        flags: [
          { name: "--message", shorthand: "-m", description: "Approval message" },
        ],
      },
      {
        name: "spec diff",
        description: "Compare two specification versions to see changes.",
        usage: "specular spec diff [version1] [version2]",
        examples: [
          "specular spec diff",
          "specular spec diff v1.0 v1.1",
        ],
        flags: [],
      },
      {
        name: "spec edit",
        description: "Edit the current specification in your default editor.",
        usage: "specular spec edit",
        examples: ["specular spec edit"],
        flags: [],
      },
      {
        name: "spec lock",
        description: "Generate a SpecLock file with cryptographic hashes for the specification.",
        usage: "specular spec lock [flags]",
        examples: [
          "specular spec lock",
          "specular spec lock --output spec.lock.json",
        ],
        flags: [
          { name: "--output", shorthand: "-o", description: "Output lock file path" },
        ],
      },
    ],
  },
  {
    id: "planning",
    name: "Planning",
    icon: "map",
    description: "Generate, review, and manage execution plans from specifications",
    commands: [
      {
        name: "plan create",
        description: "Create an execution plan from a specification. Breaks down features into tasks with dependencies and routing decisions.",
        usage: "specular plan create [flags]",
        examples: [
          "specular plan create",
          "specular plan create --feature feat-auth",
          "specular plan create --in spec.yaml --out plan.json",
        ],
        flags: [
          { name: "--in", shorthand: "-i", description: "Input spec file", default: ".specular/spec.yaml" },
          { name: "--out", shorthand: "-o", description: "Output plan file", default: "plan.json" },
          { name: "--lock", description: "Input SpecLock file", default: ".specular/spec.lock.json" },
          { name: "--feature", description: "Generate plan for specific feature ID" },
          { name: "--estimate", description: "Estimate task complexity", default: "true" },
        ],
      },
      {
        name: "plan review",
        description: "Interactively review an execution plan before building.",
        usage: "specular plan review [flags]",
        examples: [
          "specular plan review",
          "specular plan review --plan custom-plan.json",
        ],
        flags: [
          { name: "--plan", description: "Plan file to review", default: "plan.json" },
        ],
      },
      {
        name: "plan validate",
        description: "Validate plan structure and dependencies without executing.",
        usage: "specular plan validate [flags]",
        examples: [
          "specular plan validate",
          "specular plan validate --strict",
        ],
        flags: [
          { name: "--strict", description: "Enable strict validation" },
        ],
      },
      {
        name: "plan visualize",
        description: "Visualize the execution plan as a dependency graph.",
        usage: "specular plan visualize [flags]",
        examples: [
          "specular plan visualize",
          "specular plan visualize --output graph.svg",
        ],
        flags: [
          { name: "--output", shorthand: "-o", description: "Output file path" },
          { name: "--format", description: "Output format (svg, png, dot)" },
        ],
      },
      {
        name: "plan explain",
        description: "Explain routing decisions for a specific plan step.",
        usage: "specular plan explain <step-id>",
        examples: [
          "specular plan explain task-1",
          "specular plan explain --all",
        ],
        flags: [
          { name: "--all", description: "Explain all routing decisions" },
        ],
      },
    ],
  },
  {
    id: "building",
    name: "Building",
    icon: "hammer",
    description: "Execute, verify, and approve builds with policy enforcement",
    commands: [
      {
        name: "build run",
        description: "Execute the build process in a Docker sandbox with strict policy enforcement. All execution passes through guardrail checks including Docker-only enforcement, linting, testing, and security scanning.",
        usage: "specular build run [flags]",
        examples: [
          "specular build run",
          "specular build run --feature feat-auth",
          "specular build run --dry-run",
          "specular build run --resume",
        ],
        flags: [
          { name: "--plan", description: "Plan file to execute", default: "plan.json" },
          { name: "--policy", description: "Policy file for enforcement", default: ".specular/policy.yaml" },
          { name: "--feature", description: "Execute build for specific feature ID" },
          { name: "--dry-run", description: "Show what would be executed without running" },
          { name: "--resume", description: "Resume from previous checkpoint" },
          { name: "--checkpoint-id", description: "Checkpoint ID (auto-generated if not provided)" },
          { name: "--keep-checkpoint", description: "Keep checkpoint after successful completion" },
          { name: "--fail-on", description: "Fail on conditions (comma-separated: drift,lint,test,security)" },
          { name: "--enable-cache", description: "Enable Docker image caching", default: "true" },
          { name: "--verbose", description: "Verbose output" },
        ],
      },
      {
        name: "build verify",
        description: "Run lint, tests, and policy checks without full execution.",
        usage: "specular build verify [flags]",
        examples: [
          "specular build verify",
          "specular build verify --skip-lint",
        ],
        flags: [
          { name: "--skip-lint", description: "Skip linting checks" },
          { name: "--skip-tests", description: "Skip test execution" },
          { name: "--skip-security", description: "Skip security scanning" },
        ],
      },
      {
        name: "build approve",
        description: "Approve build results for deployment.",
        usage: "specular build approve [flags]",
        examples: [
          "specular build approve",
          "specular build approve --message 'Ready for staging'",
        ],
        flags: [
          { name: "--message", shorthand: "-m", description: "Approval message" },
        ],
      },
      {
        name: "build explain",
        description: "Show logs and routing decisions for the last build.",
        usage: "specular build explain [flags]",
        examples: [
          "specular build explain",
          "specular build explain --task task-3",
        ],
        flags: [
          { name: "--task", description: "Show explanation for specific task" },
        ],
      },
    ],
  },
  {
    id: "autonomous",
    name: "Autonomous Mode",
    icon: "robot",
    description: "AI-driven autonomous execution from goal to working code",
    commands: [
      {
        name: "auto",
        description: "Run Specular in autonomous agent mode. Provide a natural language goal and Specular will generate a spec, create a plan, show approval gate, and execute with policy enforcement.",
        usage: "specular auto <goal> [flags]",
        examples: [
          'specular auto "Build a REST API for user management"',
          'specular auto --profile ci "Create a React dashboard"',
          'specular auto --profile strict --dry-run "Add authentication"',
          'specular auto --scope feature:feat-1 "Execute only feature 1"',
          "specular auto --list-profiles",
        ],
        flags: [
          { name: "--profile", shorthand: "-p", description: "Profile to use (default, ci, strict, or custom)", default: "default" },
          { name: "--dry-run", description: "Generate spec and plan but don't execute" },
          { name: "--no-approval", description: "Skip approval gate (auto-approve plan)" },
          { name: "--scope", shorthand: "-s", description: "Filter execution scope (can be used multiple times)" },
          { name: "--include-dependencies", description: "Include dependencies of scoped tasks", default: "true" },
          { name: "--max-steps", description: "Maximum number of workflow steps (0 = use profile default)" },
          { name: "--max-cost", description: "Maximum cost in USD for entire workflow (0 = use profile default)" },
          { name: "--max-cost-per-task", description: "Maximum cost in USD per task (0 = use profile default)" },
          { name: "--max-retries", description: "Maximum retries per failed task (0 = use profile default)" },
          { name: "--timeout", description: "Timeout in minutes for entire workflow (0 = use profile default)" },
          { name: "--output", shorthand: "-o", description: "Output directory to save spec and plan files" },
          { name: "--json", description: "Output results in JSON format (for CI/CD integration)" },
          { name: "--tui", description: "Enable interactive TUI mode" },
          { name: "--attest", description: "Generate cryptographic attestation of workflow execution" },
          { name: "--save-patches", description: "Save patches for each step to enable rollback" },
          { name: "--trace", description: "Enable detailed trace logging to ~/.specular/logs" },
          { name: "--list-profiles", description: "List available profiles and exit" },
          { name: "--resume", description: "Resume from checkpoint (e.g., auto-1762811730)" },
        ],
      },
      {
        name: "auto resume",
        description: "Resume a paused autonomous session from checkpoint.",
        usage: "specular auto resume <session-id>",
        examples: [
          "specular auto resume auto-1762811730",
        ],
        flags: [],
      },
      {
        name: "auto rollback",
        description: "Rollback changes made by autonomous mode.",
        usage: "specular auto rollback <session-id> [flags]",
        examples: [
          "specular auto rollback auto-1762811730",
          "specular auto rollback auto-1762811730 --to-step 3",
        ],
        flags: [
          { name: "--to-step", description: "Rollback to specific step number" },
        ],
      },
      {
        name: "auto verify",
        description: "Verify cryptographic attestation of workflow execution.",
        usage: "specular auto verify <attestation-file>",
        examples: [
          "specular auto verify attestation.json",
        ],
        flags: [],
      },
      {
        name: "auto history",
        description: "View autonomous session history and logs.",
        usage: "specular auto history [flags]",
        examples: [
          "specular auto history",
          "specular auto history --session auto-1762811730",
        ],
        flags: [
          { name: "--session", description: "Show details for specific session" },
          { name: "--limit", description: "Number of sessions to show", default: "10" },
        ],
      },
      {
        name: "auto explain",
        description: "Explain reasoning for autonomous session steps.",
        usage: "specular auto explain <session-id> [flags]",
        examples: [
          "specular auto explain auto-1762811730",
          "specular auto explain auto-1762811730 --step 3",
        ],
        flags: [
          { name: "--step", description: "Explain specific step" },
        ],
      },
    ],
  },
  {
    id: "drift",
    name: "Drift Detection",
    icon: "warning",
    description: "Detect and manage drift between plan and repository state",
    commands: [
      {
        name: "drift check",
        description: "Compare the current repository state with the execution plan to detect drift. Checks file hashes, missing/extra files, and uncommitted changes.",
        usage: "specular drift check [flags]",
        examples: [
          "specular drift check",
          "specular drift check --plan custom-plan.json",
        ],
        flags: [
          { name: "--plan", description: "Plan file to check for drift", default: "plan.json" },
        ],
      },
      {
        name: "drift approve",
        description: "Approve detected drift to continue with modified state.",
        usage: "specular drift approve [flags]",
        examples: [
          "specular drift approve",
          "specular drift approve --message 'Approved manual changes'",
        ],
        flags: [
          { name: "--message", shorthand: "-m", description: "Approval message" },
        ],
      },
    ],
  },
  {
    id: "providers",
    name: "Providers",
    icon: "cloud",
    description: "Manage AI providers for various tasks",
    commands: [
      {
        name: "provider list",
        description: "List all configured AI providers with their status and capabilities.",
        usage: "specular provider list [flags]",
        examples: [
          "specular provider list",
          "specular provider list --format json",
        ],
        flags: [],
      },
      {
        name: "provider add",
        description: "Add a new AI provider to configuration.",
        usage: "specular provider add <name> [flags]",
        examples: [
          "specular provider add openai",
          "specular provider add ollama --local",
        ],
        flags: [
          { name: "--local", description: "Mark as local provider" },
          { name: "--api-key", description: "API key (or use environment variable)" },
        ],
      },
      {
        name: "provider remove",
        description: "Remove a provider from configuration.",
        usage: "specular provider remove <name>",
        examples: [
          "specular provider remove openai",
        ],
        flags: [],
      },
      {
        name: "provider doctor",
        description: "Check provider health and configuration.",
        usage: "specular provider doctor [flags]",
        examples: [
          "specular provider doctor",
          "specular provider doctor --provider ollama",
        ],
        flags: [
          { name: "--provider", description: "Check specific provider only" },
        ],
      },
      {
        name: "provider init",
        description: "Initialize provider configuration with interactive setup.",
        usage: "specular provider init [flags]",
        examples: [
          "specular provider init",
        ],
        flags: [],
      },
    ],
  },
  {
    id: "routing",
    name: "Routing",
    icon: "route",
    description: "Control how Specular selects AI models for tasks",
    commands: [
      {
        name: "route list",
        description: "List all available models and providers with costs and capabilities.",
        usage: "specular route list [flags]",
        examples: [
          "specular route list",
          "specular route list --format json",
        ],
        flags: [],
      },
      {
        name: "route override",
        description: "Override provider selection for the current session.",
        usage: "specular route override <provider>",
        examples: [
          "specular route override anthropic",
          "specular route override ollama",
        ],
        flags: [],
      },
      {
        name: "route explain",
        description: "Explain routing logic and model selection for a task type.",
        usage: "specular route explain <task-type>",
        examples: [
          "specular route explain codegen",
          "specular route explain review",
        ],
        flags: [],
      },
    ],
  },
  {
    id: "governance",
    name: "Governance",
    icon: "shield",
    description: "Initialize and manage governance infrastructure",
    commands: [
      {
        name: "governance init",
        description: "Initialize governance workspace with policies, approvals, and compliance controls.",
        usage: "specular governance init [flags]",
        examples: [
          "specular governance init",
          "specular governance init --level L3",
        ],
        flags: [
          { name: "--level", description: "Governance level (L2, L3, L4)" },
        ],
      },
      {
        name: "governance doctor",
        description: "Validate governance environment and check for issues.",
        usage: "specular governance doctor [flags]",
        examples: [
          "specular governance doctor",
        ],
        flags: [],
      },
      {
        name: "governance status",
        description: "Show governance health overview and compliance status.",
        usage: "specular governance status [flags]",
        examples: [
          "specular governance status",
        ],
        flags: [],
      },
    ],
  },
  {
    id: "policy",
    name: "Policy",
    icon: "lock",
    description: "Create, validate, and manage governance policies",
    commands: [
      {
        name: "policy init",
        description: "Create a policies.yaml template with default rules.",
        usage: "specular policy init [flags]",
        examples: [
          "specular policy init",
        ],
        flags: [],
      },
      {
        name: "policy validate",
        description: "Validate policy definitions for correctness.",
        usage: "specular policy validate [flags]",
        examples: [
          "specular policy validate",
          "specular policy validate --strict",
        ],
        flags: [
          { name: "--strict", description: "Enable strict validation" },
        ],
      },
      {
        name: "policy approve",
        description: "Approve current policies with cryptographic signature.",
        usage: "specular policy approve [flags]",
        examples: [
          "specular policy approve",
          "specular policy approve --message 'Approved for production'",
        ],
        flags: [
          { name: "--message", shorthand: "-m", description: "Approval message" },
        ],
      },
      {
        name: "policy list",
        description: "List all defined policies and their status.",
        usage: "specular policy list [flags]",
        examples: [
          "specular policy list",
        ],
        flags: [],
      },
      {
        name: "policy diff",
        description: "Show policy changes since last approval.",
        usage: "specular policy diff [flags]",
        examples: [
          "specular policy diff",
        ],
        flags: [],
      },
    ],
  },
];
