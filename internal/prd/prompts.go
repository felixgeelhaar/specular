package prd

import (
	"fmt"
	"regexp"
	"strings"
)

// buildSystemPrompt creates the system prompt for PRD-to-spec conversion
func buildSystemPrompt() string {
	return `You are an expert technical product manager and software architect. Your task is to convert Product Requirements Documents (PRDs) written in markdown into structured product specifications in JSON format.

Your goal is to extract and organize information from the PRD into a well-structured specification that can be used for software development planning and execution.

Guidelines:
1. Extract the product name, goals, and features from the PRD
2. Create unique feature IDs (feat-001, feat-002, etc.)
3. Assign priorities (P0 for critical, P1 for important, P2 for nice-to-have)
4. Extract success criteria and trace references
5. Identify non-functional requirements (performance, security, scalability, availability)
6. Create acceptance criteria and milestones
7. For API-related features, extract endpoint information

Output Requirements:
- Return ONLY valid JSON matching the ProductSpec structure
- Do NOT include markdown formatting or explanations
- Use proper JSON syntax with quoted strings
- Ensure all required fields are populated
- Make IDs sequential and consistent`
}

// buildUserPrompt creates the user prompt with the PRD content
func buildUserPrompt(prdContent string) string {
	return fmt.Sprintf(`Convert the following PRD into a structured JSON specification.

PRD Content:
%s

Output the specification as JSON with this exact structure:
{
  "product": "Product Name",
  "goals": ["Goal 1", "Goal 2"],
  "features": [
    {
      "id": "feat-001",
      "title": "Feature Title",
      "desc": "Detailed description",
      "priority": "P0",
      "api": [
        {
          "method": "HTTP Method or CLI",
          "path": "endpoint or command",
          "request": "request format",
          "response": "response format"
        }
      ],
      "success": ["Success criterion 1"],
      "trace": ["PRD-section-X"]
    }
  ],
  "non_functional": {
    "performance": ["Performance requirement 1"],
    "security": ["Security requirement 1"],
    "scalability": ["Scalability requirement 1"],
    "availability": ["Availability requirement 1"]
  },
  "acceptance": ["Acceptance criterion 1"],
  "milestones": [
    {
      "id": "m1",
      "name": "Milestone Name",
      "feature_ids": ["feat-001"],
      "target_date": "2025-01",
      "description": "Milestone description"
    }
  ]
}

Important:
- Return ONLY the JSON, no additional text or markdown
- Ensure all strings are properly quoted
- Use null for optional fields if not found in PRD
- Make feature IDs sequential (feat-001, feat-002, etc.)
- Assign P0 to critical features, P1 to important, P2 to nice-to-have
- Extract trace references from the PRD section structure`, prdContent)
}

// extractJSONFromMarkdown attempts to extract JSON from markdown code blocks
func extractJSONFromMarkdown(content string) string {
	// Try to find JSON in markdown code blocks
	jsonBlockRegex := regexp.MustCompile("```(?:json)?\\s*\\n([\\s\\S]*?)```")
	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to find JSON without code blocks (look for { ... })
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	// Find the matching closing brace
	braceCount := 0
	for i := start; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return content[start : i+1]
			}
		}
	}

	return ""
}
