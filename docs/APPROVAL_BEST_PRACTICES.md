# Approval Best Practices

This guide defines the preferred approval workflow for Specular CLI usage. The goal is a single, auditable path for governance approvals and clear separation from domain-specific signing or validation steps.

## Principles

1. Use a single canonical entry point
   - Use `specular approve <resource-id>` for governance approvals.
   - Reserve specialized commands (for example, `specular bundle approve`) for cryptographic signing or domain-specific workflows.

2. Always include a justification
   - Provide `--message` on every approval.
   - The message becomes the audit note for later review.

3. Approve immutable IDs
   - Use stable resource IDs:
     - `bundle-<id>`
     - `drift-<id>`
     - `policy-<id>`
     - `plan-<id>`
     - `exception-<id>` (controlled exception trail)

4. Keep approvals scoped and explicit
   - Approve only the resource that was reviewed.
   - Avoid broad or implicit approvals.
   - For exceptions, always set `--reason` and prefer `--scope`, `--policy`, and `--expires`.

5. Prefer read-only checks before approval
   - Use `specular approvals pending` to see what requires action.
   - Use `specular eval drift` to inspect drift before approving.
   - Use `specular approvals show` / `specular explain` to inspect the trail after DENY.

## Canonical Commands

Approve a resource:

```bash
specular approve drift-abc123 --message "Emergency hotfix reviewed"
```

List approvals:

```bash
specular approvals list
specular approvals list --json
```

Show one record (AI CHANGE RECORD style):

```bash
specular approvals show
specular approvals show exception-EX-192
```

Record a controlled exception. An open, non-expired exception can **soft-ALLOW**
a matching gate DENY when `--policy` / `--scope` binds to that deny (drift
finding code/path, failed policy check, risk category/level, or provenance /
APP / governed bind tokens such as `--policy provenance`). Unmatched
exceptions stay advisory. Prefer short `--expires`; permanent silent
suppression is discouraged.

```bash
specular approve exception-EX-192 \
  --reason "Emergency auth hotfix" \
  --scope "internal/auth/**" \
  --policy drift \
  --expires 7d \
  --message "Security on-call approved"
```

Show pending approvals:

```bash
specular approvals pending
```

## When to use specialized commands

Use specialized commands when the workflow requires cryptographic signatures or domain-specific checks:

- `specular bundle approve` for bundle signatures (SSH/GPG)
- `specular policy approve` for policy hash approvals
- `specular build approve` for local build manifest approval markers
- `specular spec approve` for local spec approval markers

These commands are not replacements for governance approvals and should be treated as local or domain-specific steps.
