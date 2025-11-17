package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for Specular
type Metrics struct {
	// Command execution metrics
	CommandExecutions *prometheus.CounterVec
	CommandDuration   *prometheus.HistogramVec
	CommandErrors     *prometheus.CounterVec

	// Provider operation metrics
	ProviderCalls   *prometheus.CounterVec
	ProviderLatency *prometheus.HistogramVec
	ProviderErrors  *prometheus.CounterVec
	ProviderCost    *prometheus.CounterVec

	// Spec generation metrics
	SpecGenerations *prometheus.CounterVec
	SpecDuration    *prometheus.HistogramVec
	SpecErrors      *prometheus.CounterVec

	// Plan generation metrics
	PlanGenerations  *prometheus.CounterVec
	PlanDuration     *prometheus.HistogramVec
	PlanFeatureCount *prometheus.HistogramVec
	PlanTaskCount    *prometheus.HistogramVec
	PlanErrors       *prometheus.CounterVec

	// Task execution metrics
	TaskExecutions *prometheus.CounterVec
	TaskDuration   *prometheus.HistogramVec
	TaskErrors     *prometheus.CounterVec

	// Docker operation metrics
	ImagePulls        *prometheus.CounterVec
	ImagePullDuration *prometheus.HistogramVec
	ImagePullErrors   *prometheus.CounterVec
	CacheHits         *prometheus.CounterVec
	CacheMisses       *prometheus.CounterVec

	// Policy check metrics
	PolicyChecks     *prometheus.CounterVec
	PolicyViolations *prometheus.CounterVec
	PolicyDuration   *prometheus.HistogramVec

	// Drift detection metrics
	DriftDetections *prometheus.CounterVec
	DriftFound      *prometheus.CounterVec

	// Autonomous mode metrics
	AutoWorkflows       *prometheus.CounterVec
	AutoSteps           *prometheus.CounterVec
	AutoStepDuration    *prometheus.HistogramVec
	AutoApprovals       *prometheus.CounterVec
	AutoApprovalLatency *prometheus.HistogramVec

	// Interview metrics
	InterviewSessions  *prometheus.CounterVec
	InterviewQuestions *prometheus.CounterVec
	InterviewDuration  *prometheus.HistogramVec

	// Error metrics (by error code from structured errors)
	Errors *prometheus.CounterVec
}

// NewMetrics creates a new Metrics instance with all metrics registered
func NewMetrics(registry prometheus.Registerer) *Metrics {
	factory := promauto.With(registry)

	return &Metrics{
		// Command metrics
		CommandExecutions: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_command_executions_total",
				Help: "Total number of command executions",
			},
			[]string{"command", "success"},
		),
		CommandDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_command_duration_seconds",
				Help:    "Command execution duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"command"},
		),
		CommandErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_command_errors_total",
				Help: "Total number of command errors",
			},
			[]string{"command", "error_code"},
		),

		// Provider metrics
		ProviderCalls: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_provider_calls_total",
				Help: "Total number of AI provider API calls",
			},
			[]string{"provider", "model", "success"},
		),
		ProviderLatency: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_provider_latency_seconds",
				Help:    "AI provider API call latency in seconds",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"provider", "model"},
		),
		ProviderErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_provider_errors_total",
				Help: "Total number of AI provider errors",
			},
			[]string{"provider", "model", "error_type"},
		),
		ProviderCost: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_provider_cost_tokens_total",
				Help: "Total token cost for AI provider calls",
			},
			[]string{"provider", "model", "token_type"},
		),

		// Spec generation metrics
		SpecGenerations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_spec_generations_total",
				Help: "Total number of spec generations",
			},
			[]string{"success"},
		),
		SpecDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_spec_duration_seconds",
				Help:    "Spec generation duration in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0},
			},
			[]string{},
		),
		SpecErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_spec_errors_total",
				Help: "Total number of spec generation errors",
			},
			[]string{"error_code"},
		),

		// Plan generation metrics
		PlanGenerations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_plan_generations_total",
				Help: "Total number of plan generations",
			},
			[]string{"success"},
		),
		PlanDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_plan_duration_seconds",
				Help:    "Plan generation duration in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0},
			},
			[]string{},
		),
		PlanFeatureCount: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_plan_feature_count",
				Help:    "Number of features in generated plans",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{},
		),
		PlanTaskCount: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_plan_task_count",
				Help:    "Number of tasks in generated plans",
				Buckets: []float64{1, 5, 10, 20, 50, 100, 200},
			},
			[]string{},
		),
		PlanErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_plan_errors_total",
				Help: "Total number of plan generation errors",
			},
			[]string{"error_code"},
		),

		// Task execution metrics
		TaskExecutions: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_task_executions_total",
				Help: "Total number of task executions",
			},
			[]string{"success"},
		),
		TaskDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_task_duration_seconds",
				Help:    "Task execution duration in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, 600.0},
			},
			[]string{"task_type"},
		),
		TaskErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_task_errors_total",
				Help: "Total number of task execution errors",
			},
			[]string{"task_type", "error_code"},
		),

		// Docker metrics
		ImagePulls: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_image_pulls_total",
				Help: "Total number of Docker image pulls",
			},
			[]string{"image", "success"},
		),
		ImagePullDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_image_pull_duration_seconds",
				Help:    "Docker image pull duration in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0},
			},
			[]string{"image"},
		),
		ImagePullErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_image_pull_errors_total",
				Help: "Total number of Docker image pull errors",
			},
			[]string{"image", "error_type"},
		),
		CacheHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_cache_hits_total",
				Help: "Total number of Docker image cache hits",
			},
			[]string{"image"},
		),
		CacheMisses: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_cache_misses_total",
				Help: "Total number of Docker image cache misses",
			},
			[]string{"image"},
		),

		// Policy metrics
		PolicyChecks: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_policy_checks_total",
				Help: "Total number of policy checks",
			},
			[]string{"policy_type", "result"},
		),
		PolicyViolations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_policy_violations_total",
				Help: "Total number of policy violations",
			},
			[]string{"policy_type", "severity"},
		),
		PolicyDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_policy_duration_seconds",
				Help:    "Policy check duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"policy_type"},
		),

		// Drift detection metrics
		DriftDetections: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_drift_detections_total",
				Help: "Total number of drift detections run",
			},
			[]string{"drift_type"},
		),
		DriftFound: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_drift_found_total",
				Help: "Total number of times drift was found",
			},
			[]string{"drift_type"},
		),

		// Autonomous mode metrics
		AutoWorkflows: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_auto_workflows_total",
				Help: "Total number of autonomous mode workflows",
			},
			[]string{"success"},
		),
		AutoSteps: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_auto_steps_total",
				Help: "Total number of autonomous mode steps",
			},
			[]string{"step_type", "success"},
		),
		AutoStepDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_auto_step_duration_seconds",
				Help:    "Autonomous mode step duration in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0},
			},
			[]string{"step_type"},
		),
		AutoApprovals: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_auto_approvals_total",
				Help: "Total number of autonomous mode approval requests",
			},
			[]string{"approved"},
		),
		AutoApprovalLatency: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_auto_approval_latency_seconds",
				Help:    "Time taken for approval decisions in seconds",
				Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 300.0, 600.0},
			},
			[]string{},
		),

		// Interview metrics
		InterviewSessions: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_interview_sessions_total",
				Help: "Total number of interview sessions",
			},
			[]string{"preset", "success"},
		),
		InterviewQuestions: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_interview_questions_total",
				Help: "Total number of interview questions asked",
			},
			[]string{"preset"},
		),
		InterviewDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "specular_interview_duration_seconds",
				Help:    "Interview session duration in seconds",
				Buckets: []float64{10.0, 30.0, 60.0, 120.0, 300.0, 600.0},
			},
			[]string{"preset"},
		),

		// Error metrics (by structured error code)
		Errors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "specular_errors_total",
				Help: "Total number of errors by error code",
			},
			[]string{"error_code", "component"},
		),
	}
}
