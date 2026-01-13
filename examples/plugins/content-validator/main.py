#!/usr/bin/env python3
"""Content Validator - A Specular validator plugin.

This plugin validates content against configurable rules including:
- Length constraints (min/max)
- Required keywords
- Forbidden patterns (regex)
"""

import json
import re
import sys
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

VERSION = "1.0.0"


@dataclass
class ValidationIssue:
    """Represents a validation issue."""

    severity: str  # error, warning, info
    message: str
    rule: str
    line: Optional[int] = None
    column: Optional[int] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "severity": self.severity,
            "message": self.message,
            "rule": self.rule,
        }
        if self.line is not None:
            result["line"] = self.line
        if self.column is not None:
            result["column"] = self.column
        return result


@dataclass
class Config:
    """Plugin configuration."""

    max_length: int = 10000
    min_length: int = 0
    required_keywords: List[str] = field(default_factory=list)
    forbidden_patterns: List[str] = field(default_factory=list)
    severity_threshold: str = "warning"

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Config":
        """Create Config from dictionary."""
        config = cls()

        if "max_length" in data:
            config.max_length = int(data["max_length"])

        if "min_length" in data:
            config.min_length = int(data["min_length"])

        if "required_keywords" in data and data["required_keywords"]:
            keywords = data["required_keywords"]
            if isinstance(keywords, str):
                config.required_keywords = [k.strip() for k in keywords.split(",") if k.strip()]
            elif isinstance(keywords, list):
                config.required_keywords = keywords

        if "forbidden_patterns" in data and data["forbidden_patterns"]:
            patterns = data["forbidden_patterns"]
            if isinstance(patterns, str):
                config.forbidden_patterns = [p.strip() for p in patterns.split(",") if p.strip()]
            elif isinstance(patterns, list):
                config.forbidden_patterns = patterns

        if "severity_threshold" in data:
            config.severity_threshold = data["severity_threshold"]

        return config


def health_check() -> Dict[str, str]:
    """Return health status."""
    return {
        "status": "healthy",
        "version": VERSION,
        "name": "content-validator",
    }


def validate_content(content: str, config: Config) -> List[ValidationIssue]:
    """Validate content against configured rules."""
    issues: List[ValidationIssue] = []

    # Check minimum length
    if len(content) < config.min_length:
        issues.append(ValidationIssue(
            severity="error",
            message=f"Content too short: {len(content)} characters, minimum is {config.min_length}",
            rule="min_length",
        ))

    # Check maximum length
    if len(content) > config.max_length:
        issues.append(ValidationIssue(
            severity="error",
            message=f"Content too long: {len(content)} characters, maximum is {config.max_length}",
            rule="max_length",
        ))

    # Check required keywords
    for keyword in config.required_keywords:
        if keyword.lower() not in content.lower():
            issues.append(ValidationIssue(
                severity="warning",
                message=f"Required keyword missing: '{keyword}'",
                rule="required_keyword",
            ))

    # Check forbidden patterns
    for pattern in config.forbidden_patterns:
        try:
            matches = list(re.finditer(pattern, content, re.IGNORECASE))
            for match in matches:
                # Calculate line number
                line_num = content[:match.start()].count('\n') + 1
                issues.append(ValidationIssue(
                    severity="error",
                    message=f"Forbidden pattern found: '{pattern}' matches '{match.group()}'",
                    rule="forbidden_pattern",
                    line=line_num,
                ))
        except re.error as e:
            issues.append(ValidationIssue(
                severity="warning",
                message=f"Invalid regex pattern '{pattern}': {e}",
                rule="invalid_pattern",
            ))

    return issues


def filter_by_severity(issues: List[ValidationIssue], threshold: str) -> List[ValidationIssue]:
    """Filter issues by severity threshold."""
    severity_order = {"info": 0, "warning": 1, "error": 2}
    threshold_level = severity_order.get(threshold, 1)

    return [
        issue for issue in issues
        if severity_order.get(issue.severity, 0) >= threshold_level
    ]


def handle_validate(request: Dict[str, Any]) -> Dict[str, Any]:
    """Handle validation request."""
    data = request.get("data", {})
    raw_config = request.get("config", {})

    content = data.get("content", "")
    if not content:
        return {
            "valid": False,
            "messages": [{
                "severity": "error",
                "message": "No content provided for validation",
                "rule": "content_required",
            }],
        }

    config = Config.from_dict(raw_config)
    issues = validate_content(content, config)
    filtered_issues = filter_by_severity(issues, config.severity_threshold)

    # Content is valid if there are no errors
    has_errors = any(issue.severity == "error" for issue in filtered_issues)

    return {
        "valid": not has_errors,
        "messages": [issue.to_dict() for issue in filtered_issues],
    }


def handle_request(request: Dict[str, Any]) -> Dict[str, Any]:
    """Handle incoming plugin request."""
    action = request.get("action", "")

    if action == "health":
        return {"success": True, "result": health_check()}

    if action == "validate":
        try:
            result = handle_validate(request)
            return {"success": True, "result": result}
        except Exception as e:
            return {"success": False, "error": f"validation: {e}"}

    return {"success": False, "error": f"unknown action: {action}"}


def main() -> None:
    """Main entry point."""
    for line in sys.stdin:
        try:
            request = json.loads(line.strip())
            response = handle_request(request)
        except json.JSONDecodeError as e:
            response = {"success": False, "error": f"json: {e}"}
        except Exception as e:
            response = {"success": False, "error": f"internal: {e}"}

        print(json.dumps(response), flush=True)


if __name__ == "__main__":
    main()
