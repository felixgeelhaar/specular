package interview

// GetPresets returns all available interview presets
func GetPresets() map[string]Preset {
	return map[string]Preset{
		"web-app":       WebAppPreset(),
		"api-service":   APIServicePreset(),
		"cli-tool":      CLIToolPreset(),
		"microservice":  MicroservicePreset(),
		"data-pipeline": DataPipelinePreset(),
	}
}

// WebAppPreset returns questions for a web application
func WebAppPreset() Preset {
	return Preset{
		Name:        "web-app",
		Description: "Web application (SPA, dashboard, admin panel)",
		Questions: []Question{
			{
				ID:          "product-name",
				Type:        QuestionTypeText,
				Text:        "What is the name of your product?",
				Description: "A concise name for your web application",
				Required:    true,
			},
			{
				ID:          "product-purpose",
				Type:        QuestionTypeText,
				Text:        "What is the main purpose of this application?",
				Description: "Describe what problem it solves or what value it provides",
				Required:    true,
			},
			{
				ID:          "target-users",
				Type:        QuestionTypeText,
				Text:        "Who are the primary users?",
				Description: "Describe your target audience (e.g., 'internal team members', 'enterprise customers')",
				Required:    true,
			},
			{
				ID:          "core-features",
				Type:        QuestionTypeMulti,
				Text:        "What are the core features? (Enter each on a new line)",
				Description: "List 3-7 essential features that define your MVP",
				Required:    true,
			},
			{
				ID:       "auth-required",
				Type:     QuestionTypeYesNo,
				Text:     "Does this application require user authentication?",
				Required: true,
			},
			{
				ID:       "auth-type",
				Type:     QuestionTypeChoice,
				Text:     "What type of authentication?",
				Choices:  []string{"Email/Password", "OAuth (Google, GitHub, etc.)", "SSO/SAML", "Magic Link"},
				Required: true,
				SkipIf:   "auth-required=no",
			},
			{
				ID:       "data-storage",
				Type:     QuestionTypeChoice,
				Text:     "What type of data storage do you need?",
				Choices:  []string{"SQL Database (PostgreSQL, MySQL)", "NoSQL (MongoDB, DynamoDB)", "Both SQL and NoSQL", "File storage only", "No persistence needed"},
				Required: true,
			},
			{
				ID:       "ui-framework",
				Type:     QuestionTypeChoice,
				Text:     "Preferred frontend framework?",
				Choices:  []string{"React", "Vue", "Angular", "Svelte", "No preference"},
				Required: false,
			},
			{
				ID:       "real-time",
				Type:     QuestionTypeYesNo,
				Text:     "Do you need real-time updates (WebSockets, SSE)?",
				Required: true,
			},
			{
				ID:       "api-integration",
				Type:     QuestionTypeYesNo,
				Text:     "Will this integrate with external APIs?",
				Required: true,
			},
			{
				ID:       "deployment-target",
				Type:     QuestionTypeChoice,
				Text:     "Where will this be deployed?",
				Choices:  []string{"Cloud (AWS, GCP, Azure)", "On-premise", "Hybrid", "Undecided"},
				Required: true,
			},
			{
				ID:          "performance-requirements",
				Type:        QuestionTypeText,
				Text:        "Any specific performance requirements?",
				Description: "E.g., 'Handle 10,000 concurrent users', 'Page load < 2s', or 'None'",
				Required:    false,
			},
			{
				ID:          "success-criteria",
				Type:        QuestionTypeMulti,
				Text:        "How will you measure success? (Enter each criterion on a new line)",
				Description: "Define 2-4 measurable success criteria",
				Required:    true,
			},
		},
	}
}

// APIServicePreset returns questions for an API service
func APIServicePreset() Preset {
	return Preset{
		Name:        "api-service",
		Description: "REST or GraphQL API service",
		Questions: []Question{
			{
				ID:       "product-name",
				Type:     QuestionTypeText,
				Text:     "What is the name of your API service?",
				Required: true,
			},
			{
				ID:          "api-purpose",
				Type:        QuestionTypeText,
				Text:        "What does this API do?",
				Description: "Describe the core functionality and business domain",
				Required:    true,
			},
			{
				ID:       "api-type",
				Type:     QuestionTypeChoice,
				Text:     "What type of API?",
				Choices:  []string{"RESTful API", "GraphQL", "gRPC", "WebSocket", "Mixed"},
				Required: true,
			},
			{
				ID:          "main-resources",
				Type:        QuestionTypeMulti,
				Text:        "What are the main resources/entities? (Enter each on a new line)",
				Description: "E.g., 'Users', 'Products', 'Orders', 'Payments'",
				Required:    true,
			},
			{
				ID:       "auth-method",
				Type:     QuestionTypeChoice,
				Text:     "Authentication method?",
				Choices:  []string{"JWT", "API Keys", "OAuth 2.0", "mTLS", "No authentication"},
				Required: true,
			},
			{
				ID:       "rate-limiting",
				Type:     QuestionTypeYesNo,
				Text:     "Do you need rate limiting?",
				Required: true,
			},
			{
				ID:       "data-storage",
				Type:     QuestionTypeChoice,
				Text:     "Data storage backend?",
				Choices:  []string{"PostgreSQL", "MySQL", "MongoDB", "DynamoDB", "Redis", "Multiple"},
				Required: true,
			},
			{
				ID:          "expected-throughput",
				Type:        QuestionTypeText,
				Text:        "Expected request throughput?",
				Description: "E.g., '1000 req/sec', '100K req/day', or 'Unknown'",
				Required:    false,
			},
			{
				ID:       "success-criteria",
				Type:     QuestionTypeMulti,
				Text:     "What are your success criteria? (Enter each on a new line)",
				Required: true,
			},
		},
	}
}

// CLIToolPreset returns questions for a CLI tool
func CLIToolPreset() Preset {
	return Preset{
		Name:        "cli-tool",
		Description: "Command-line interface tool",
		Questions: []Question{
			{
				ID:       "product-name",
				Type:     QuestionTypeText,
				Text:     "What is the name of your CLI tool?",
				Required: true,
			},
			{
				ID:          "tool-purpose",
				Type:        QuestionTypeText,
				Text:        "What does this tool do?",
				Description: "Describe its main purpose and use case",
				Required:    true,
			},
			{
				ID:          "target-users",
				Type:        QuestionTypeText,
				Text:        "Who will use this tool?",
				Description: "E.g., 'Developers', 'DevOps engineers', 'Data scientists'",
				Required:    true,
			},
			{
				ID:          "main-commands",
				Type:        QuestionTypeMulti,
				Text:        "What are the main commands? (Enter each on a new line)",
				Description: "E.g., 'init', 'build', 'deploy', 'status'",
				Required:    true,
			},
			{
				ID:       "config-file",
				Type:     QuestionTypeYesNo,
				Text:     "Does it use a configuration file?",
				Required: true,
			},
			{
				ID:       "config-format",
				Type:     QuestionTypeChoice,
				Text:     "Configuration file format?",
				Choices:  []string{"YAML", "JSON", "TOML", "INI", "Custom"},
				Required: true,
				SkipIf:   "config-file=no",
			},
			{
				ID:       "output-format",
				Type:     QuestionTypeChoice,
				Text:     "Primary output format?",
				Choices:  []string{"Text/Pretty", "JSON", "Table", "Multiple formats"},
				Required: true,
			},
			{
				ID:          "interactive-mode",
				Type:        QuestionTypeYesNo,
				Text:        "Does it need an interactive mode?",
				Description: "E.g., prompts, menus, or TUI",
				Required:    true,
			},
			{
				ID:       "success-criteria",
				Type:     QuestionTypeMulti,
				Text:     "Success criteria? (Enter each on a new line)",
				Required: true,
			},
		},
	}
}

// MicroservicePreset returns questions for a microservice
func MicroservicePreset() Preset {
	return Preset{
		Name:        "microservice",
		Description: "Single-purpose microservice",
		Questions: []Question{
			{
				ID:       "service-name",
				Type:     QuestionTypeText,
				Text:     "Service name?",
				Required: true,
			},
			{
				ID:          "service-responsibility",
				Type:        QuestionTypeText,
				Text:        "What is this service responsible for?",
				Description: "Single clear responsibility (following SRP)",
				Required:    true,
			},
			{
				ID:       "communication-pattern",
				Type:     QuestionTypeChoice,
				Text:     "How does it communicate with other services?",
				Choices:  []string{"Synchronous (HTTP/gRPC)", "Asynchronous (Message Queue)", "Both", "Standalone"},
				Required: true,
			},
			{
				ID:       "message-broker",
				Type:     QuestionTypeChoice,
				Text:     "Message broker?",
				Choices:  []string{"RabbitMQ", "Apache Kafka", "AWS SQS/SNS", "Google Pub/Sub", "NATS"},
				Required: true,
				SkipIf:   "communication-pattern=Synchronous (HTTP/gRPC)",
			},
			{
				ID:          "data-ownership",
				Type:        QuestionTypeYesNo,
				Text:        "Does this service own its own database?",
				Description: "Following database-per-service pattern",
				Required:    true,
			},
			{
				ID:          "observability",
				Type:        QuestionTypeMulti,
				Text:        "Required observability features? (Select multiple)",
				Description: "Logging, Metrics, Distributed Tracing, Health Checks",
				Required:    true,
			},
			{
				ID:       "success-criteria",
				Type:     QuestionTypeMulti,
				Text:     "Success criteria? (Enter each on a new line)",
				Required: true,
			},
		},
	}
}

// DataPipelinePreset returns questions for a data pipeline
func DataPipelinePreset() Preset {
	return Preset{
		Name:        "data-pipeline",
		Description: "Data processing pipeline or ETL",
		Questions: []Question{
			{
				ID:       "pipeline-name",
				Type:     QuestionTypeText,
				Text:     "Pipeline name?",
				Required: true,
			},
			{
				ID:          "pipeline-purpose",
				Type:        QuestionTypeText,
				Text:        "What data does this pipeline process?",
				Description: "Describe the data transformation or processing",
				Required:    true,
			},
			{
				ID:          "data-sources",
				Type:        QuestionTypeMulti,
				Text:        "Data sources? (Enter each on a new line)",
				Description: "E.g., 'S3 buckets', 'PostgreSQL', 'REST APIs', 'Kafka topics'",
				Required:    true,
			},
			{
				ID:          "data-destinations",
				Type:        QuestionTypeMulti,
				Text:        "Data destinations? (Enter each on a new line)",
				Description: "Where does processed data go?",
				Required:    true,
			},
			{
				ID:       "processing-mode",
				Type:     QuestionTypeChoice,
				Text:     "Processing mode?",
				Choices:  []string{"Batch (scheduled)", "Real-time (streaming)", "Both"},
				Required: true,
			},
			{
				ID:          "data-volume",
				Type:        QuestionTypeText,
				Text:        "Expected data volume?",
				Description: "E.g., '100GB/day', '1M records/hour', or 'Unknown'",
				Required:    false,
			},
			{
				ID:          "data-quality",
				Type:        QuestionTypeYesNo,
				Text:        "Do you need data quality checks?",
				Description: "Validation, schema enforcement, anomaly detection",
				Required:    true,
			},
			{
				ID:       "success-criteria",
				Type:     QuestionTypeMulti,
				Text:     "Success criteria? (Enter each on a new line)",
				Required: true,
			},
		},
	}
}
