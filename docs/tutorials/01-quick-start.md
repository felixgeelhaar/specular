# Quick Start: Your First Spec in 5 Minutes

This tutorial gets you from zero to a complete product specification in under 5 minutes.

## Prerequisites

- Specular installed (`brew install felixgeelhaar/tap/specular`)
- An AI provider API key (OpenAI, Anthropic, or Gemini)

## Step 1: Create Project Directory

```bash
mkdir my-project
cd my-project
```

## Step 2: Initialize Specular

```bash
specular init
```

This creates:
- `.specular/` directory with configuration
- `providers.yaml` for AI provider setup
- `policy.yaml` for governance rules

## Step 3: Configure AI Provider

When prompted, select your AI provider and enter your API key:

```
? Select primary AI provider:
  > OpenAI
    Anthropic
    Google (Gemini)
    Ollama (Local)

? Enter your OpenAI API key: sk-...
```

## Step 4: Generate Your Spec

### Option A: Interactive Interview (Recommended)

```bash
specular interview --tui
```

The TUI guides you through questions about:
- Product name and description
- Target users
- Core features
- Technical requirements

### Option B: From Existing PRD

If you have a PRD document:

```bash
specular spec generate --in PRD.md --out .specular/spec.yaml
```

## Step 5: Review Your Spec

Open the generated specification:

```bash
cat .specular/spec.yaml
```

You'll see a structured spec with:
- Product metadata
- Features with priorities (P0/P1/P2)
- API definitions
- Acceptance criteria

## Next Steps

Now that you have a spec, you can:

1. **Generate an execution plan:**
   ```bash
   specular plan
   ```

2. **Run a dry-run build:**
   ```bash
   specular build --dry-run
   ```

3. **Detect drift:**
   ```bash
   specular eval
   ```

## Troubleshooting

### "No provider configured"

Run `specular init` again or manually edit `.specular/providers.yaml`.

### "API key invalid"

Check your API key is correct and has sufficient credits.

### "Interview stuck"

Press `Ctrl+C` to cancel and restart with `specular interview --tui`.

## Learn More

- [Full Workflow Tutorial](./02-full-workflow.md)
- [Using Templates](./03-using-templates.md)
- [CLI Reference](../CLI_REFERENCE.md)
