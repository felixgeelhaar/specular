package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/alerting"
	"github.com/felixgeelhaar/specular/internal/slo"
	"github.com/felixgeelhaar/specular/internal/ux"
)

func init() {
	rootCmd.AddCommand(observabilityCmd)

	// Add subcommands
	observabilityCmd.AddCommand(obsStatusCmd)
	observabilityCmd.AddCommand(obsSLOCmd)
	observabilityCmd.AddCommand(obsAlertsCmd)
	observabilityCmd.AddCommand(obsMetricsCmd)

	// SLO subcommands
	obsSLOCmd.AddCommand(sloStatusCmd)
	obsSLOCmd.AddCommand(sloBurnRateCmd)
	obsSLOCmd.AddCommand(sloConfigCmd)

	// Alerts subcommands
	obsAlertsCmd.AddCommand(alertsTestCmd)
	obsAlertsCmd.AddCommand(alertsStatusCmd)
}

var observabilityCmd = &cobra.Command{
	Use:     "observability",
	Aliases: []string{"obs", "o11y"},
	Short:   "Observability and monitoring commands",
	Long: `Manage observability features including SLOs, alerts, and metrics.

Available Commands:
  status     Show overall observability status
  slo        Service Level Objective management
  alerts     Alert routing and testing
  metrics    View current metrics

Examples:
  # Check observability status
  specular observability status

  # View SLO compliance
  specular observability slo status

  # Test alert routing
  specular observability alerts test
`,
}

// ========================================
// Status Command
// ========================================

var obsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show observability status",
	Long:  "Display the current status of all observability components.",
	RunE:  runObsStatus,
}

// ObservabilityStatus represents the observability component status
type ObservabilityStatus struct {
	Logging   ComponentStatus `json:"logging"`
	Tracing   ComponentStatus `json:"tracing"`
	SLO       ComponentStatus `json:"slo"`
	Alerting  ComponentStatus `json:"alerting"`
	LogExport ComponentStatus `json:"log_export"`
}

// ComponentStatus represents the status of an observability component
type ComponentStatus struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"` // "healthy", "degraded", "disabled"
	Details string `json:"details,omitempty"`
}

func runObsStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	status := &ObservabilityStatus{
		Logging: ComponentStatus{
			Enabled: true,
			Status:  "healthy",
			Details: fmt.Sprintf("Level: %s", cfg.Logging.Level),
		},
		Tracing: ComponentStatus{
			Enabled: cfg.Telemetry.Enabled,
			Status:  getComponentStatus(cfg.Telemetry.Enabled),
			Details: getTracingDetails(cfg),
		},
		SLO: ComponentStatus{
			Enabled: cfg.Observability.SLO.Enabled,
			Status:  getComponentStatus(cfg.Observability.SLO.Enabled),
			Details: getSLODetails(cfg),
		},
		Alerting: ComponentStatus{
			Enabled: cfg.Observability.Alerting.Enabled,
			Status:  getComponentStatus(cfg.Observability.Alerting.Enabled),
			Details: getAlertingDetails(cfg),
		},
		LogExport: ComponentStatus{
			Enabled: cfg.Observability.LogExport.Type != "",
			Status:  getComponentStatus(cfg.Observability.LogExport.Type != ""),
			Details: getLogExportDetails(cfg),
		},
	}

	return outputObsStatus(cmdCtx, status)
}

func outputObsStatus(cmdCtx *CommandContext, status *ObservabilityStatus) error {
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(status)
	}

	// Text output
	fmt.Println("Observability Status")
	fmt.Println("====================")
	fmt.Printf("Logging:    %s (%s)\n", status.Logging.Status, status.Logging.Details)
	fmt.Printf("Tracing:    %s (%s)\n", status.Tracing.Status, status.Tracing.Details)
	fmt.Printf("SLO:        %s (%s)\n", status.SLO.Status, status.SLO.Details)
	fmt.Printf("Alerting:   %s (%s)\n", status.Alerting.Status, status.Alerting.Details)
	fmt.Printf("Log Export: %s (%s)\n", status.LogExport.Status, status.LogExport.Details)

	return nil
}

func getComponentStatus(enabled bool) string {
	if enabled {
		return "healthy"
	}
	return "disabled"
}

func getTracingDetails(cfg *GlobalConfig) string {
	if !cfg.Telemetry.Enabled {
		return "Distributed tracing disabled"
	}
	return fmt.Sprintf("Endpoint: %s, Sample Rate: %.2f", cfg.Telemetry.Endpoint, cfg.Telemetry.SampleRate)
}

func getSLODetails(cfg *GlobalConfig) string {
	if !cfg.Observability.SLO.Enabled {
		return "SLO tracking disabled"
	}
	if cfg.Observability.SLO.ConfigPath != "" {
		return fmt.Sprintf("Config: %s", cfg.Observability.SLO.ConfigPath)
	}
	return "Using default SLOs"
}

func getAlertingDetails(cfg *GlobalConfig) string {
	if !cfg.Observability.Alerting.Enabled {
		return "Alert routing disabled"
	}
	var providers []string
	if cfg.Observability.Alerting.PagerDuty.RoutingKey != "" {
		providers = append(providers, "pagerduty")
	}
	if cfg.Observability.Alerting.Opsgenie.APIKey != "" {
		providers = append(providers, "opsgenie")
	}
	if cfg.Observability.Alerting.Slack.WebhookURL != "" {
		providers = append(providers, "slack")
	}
	if cfg.Observability.Alerting.Webhook.URL != "" {
		providers = append(providers, "webhook")
	}
	if len(providers) == 0 {
		return "No alert destinations configured"
	}
	return fmt.Sprintf("Destinations: %v", providers)
}

func getLogExportDetails(cfg *GlobalConfig) string {
	if cfg.Observability.LogExport.Type == "" {
		return "Log export disabled"
	}
	return fmt.Sprintf("Type: %s, Endpoint: %s", cfg.Observability.LogExport.Type, cfg.Observability.LogExport.Endpoint)
}

// ========================================
// SLO Commands
// ========================================

var obsSLOCmd = &cobra.Command{
	Use:   "slo",
	Short: "Service Level Objective management",
	Long:  "Manage and monitor Service Level Objectives (SLOs).",
}

var sloStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show SLO compliance status",
	Long:  "Display the current compliance status of all configured SLOs.",
	RunE:  runSLOStatus,
}

// SLOStatusReport represents the SLO status output
type SLOStatusReport struct {
	SLOs    []SLOStatusEntry `json:"slos"`
	Summary SLOSummary       `json:"summary"`
}

// SLOStatusEntry represents a single SLO status
type SLOStatusEntry struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Target          float64 `json:"target"`
	Current         float64 `json:"current"`
	ErrorBudget     float64 `json:"error_budget_remaining"`
	BurnRate        float64 `json:"burn_rate"`
	Status          string  `json:"status"` // "healthy", "warning", "critical"
	AlertFiring     bool    `json:"alert_firing"`
	ProjectedBudget float64 `json:"projected_budget_at_window_end"`
}

// SLOSummary represents the SLO summary
type SLOSummary struct {
	Total    int `json:"total"`
	Healthy  int `json:"healthy"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

func runSLOStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	if !cfg.Observability.SLO.Enabled {
		fmt.Println("SLO tracking is not enabled. Enable it in your configuration.")
		fmt.Println("\nTo enable, add to ~/.specular/config.yaml:")
		fmt.Println("  observability:")
		fmt.Println("    slo:")
		fmt.Println("      enabled: true")
		return nil
	}

	// Load SLOs
	var slos []*slo.SLO
	if cfg.Observability.SLO.ConfigPath != "" {
		slos, err = slo.LoadSLOsFromFile(cfg.Observability.SLO.ConfigPath)
		if err != nil {
			return ux.FormatError(err, "loading SLOs from file")
		}
	} else if len(cfg.Observability.SLO.SLOs) > 0 {
		slos = cfg.Observability.SLO.SLOs
	} else {
		// Use defaults
		slos = slo.DefaultSLOs()
	}

	report := &SLOStatusReport{
		SLOs: make([]SLOStatusEntry, 0, len(slos)),
	}

	for _, s := range slos {
		entry := SLOStatusEntry{
			Name:            s.Name,
			Description:     s.Description,
			Target:          s.Target,
			Current:         0.0, // Would come from MetricsProvider
			ErrorBudget:     1.0, // 100% remaining when no data
			BurnRate:        0.0,
			Status:          "unknown",
			AlertFiring:     false,
			ProjectedBudget: 1.0,
		}
		report.SLOs = append(report.SLOs, entry)
		report.Summary.Total++
	}

	return outputSLOStatus(cmdCtx, report)
}

func outputSLOStatus(cmdCtx *CommandContext, report *SLOStatusReport) error {
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(report)
	}

	// Text output
	fmt.Println("SLO Status")
	fmt.Println("==========")
	fmt.Printf("Total: %d | Healthy: %d | Warning: %d | Critical: %d\n\n",
		report.Summary.Total, report.Summary.Healthy, report.Summary.Warning, report.Summary.Critical)

	for _, s := range report.SLOs {
		fmt.Printf("%-30s Target: %.2f%%  Current: %.2f%%  Status: %s\n",
			s.Name, s.Target*100, s.Current*100, s.Status)
	}

	return nil
}

var sloBurnRateCmd = &cobra.Command{
	Use:   "burn-rate",
	Short: "Show SLO burn rates",
	Long:  "Display the current burn rates for all SLOs across different time windows.",
	RunE:  runSLOBurnRate,
}

// BurnRateReport represents burn rate information
type BurnRateReport struct {
	SLOs []BurnRateEntry `json:"slos"`
}

// BurnRateEntry represents burn rate for a single SLO
type BurnRateEntry struct {
	Name        string  `json:"name"`
	Target      float64 `json:"target"`
	BurnRate1h  float64 `json:"burn_rate_1h"`
	BurnRate6h  float64 `json:"burn_rate_6h"`
	BurnRate24h float64 `json:"burn_rate_24h"`
	Alerting    bool    `json:"alerting"`
}

func runSLOBurnRate(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	if !cfg.Observability.SLO.Enabled {
		return fmt.Errorf("SLO tracking is not enabled")
	}

	// Load SLOs
	var slos []*slo.SLO
	if cfg.Observability.SLO.ConfigPath != "" {
		slos, err = slo.LoadSLOsFromFile(cfg.Observability.SLO.ConfigPath)
		if err != nil {
			return ux.FormatError(err, "loading SLOs")
		}
	} else {
		slos = slo.DefaultSLOs()
	}

	report := &BurnRateReport{
		SLOs: make([]BurnRateEntry, 0, len(slos)),
	}

	for _, s := range slos {
		entry := BurnRateEntry{
			Name:        s.Name,
			Target:      s.Target,
			BurnRate1h:  0.0, // Would come from actual metrics
			BurnRate6h:  0.0,
			BurnRate24h: 0.0,
			Alerting:    false,
		}
		report.SLOs = append(report.SLOs, entry)
	}

	return outputBurnRate(cmdCtx, report)
}

func outputBurnRate(cmdCtx *CommandContext, report *BurnRateReport) error {
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(report)
	}

	// Text output
	fmt.Println("SLO Burn Rates")
	fmt.Println("==============")
	fmt.Printf("%-30s %-10s %-10s %-10s %-10s\n", "Name", "1h", "6h", "24h", "Alerting")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, s := range report.SLOs {
		alertStatus := "no"
		if s.Alerting {
			alertStatus = "YES"
		}
		fmt.Printf("%-30s %-10.2f %-10.2f %-10.2f %-10s\n",
			s.Name, s.BurnRate1h, s.BurnRate6h, s.BurnRate24h, alertStatus)
	}

	return nil
}

var sloConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show SLO configuration",
	Long:  "Display the current SLO configuration and generate example configs.",
	RunE:  runSLOConfig,
}

func runSLOConfig(cmd *cobra.Command, args []string) error {
	// Print example SLO configuration
	fmt.Println(slo.ExampleSLOFile())
	return nil
}

// ========================================
// Alerts Commands
// ========================================

var obsAlertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Alert routing management",
	Long:  "Manage alert routing and test alert destinations.",
}

var alertsTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test alert routing",
	Long:  "Send a test alert to all configured destinations to verify connectivity.",
	RunE:  runAlertsTest,
}

// AlertTestResult represents the result of an alert test
type AlertTestResult struct {
	Destinations []AlertDestinationResult `json:"destinations"`
	Success      bool                     `json:"success"`
}

// AlertDestinationResult represents the test result for one destination
type AlertDestinationResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func runAlertsTest(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	if !cfg.Observability.Alerting.Enabled {
		return fmt.Errorf("alerting is not enabled")
	}

	// Build alert managers from config
	var managers []alerting.AlertManager

	if cfg.Observability.Alerting.PagerDuty.RoutingKey != "" {
		mgr, err := alerting.NewPagerDutyManager(alerting.PagerDutyConfig{
			RoutingKey: cfg.Observability.Alerting.PagerDuty.RoutingKey,
			ServiceID:  cfg.Observability.Alerting.PagerDuty.ServiceID,
		})
		if err == nil {
			managers = append(managers, mgr)
		}
	}

	if cfg.Observability.Alerting.Opsgenie.APIKey != "" {
		mgr, err := alerting.NewOpsgenieManager(alerting.OpsgenieConfig{
			APIKey: cfg.Observability.Alerting.Opsgenie.APIKey,
			Region: cfg.Observability.Alerting.Opsgenie.Region,
			TeamID: cfg.Observability.Alerting.Opsgenie.TeamID,
		})
		if err == nil {
			managers = append(managers, mgr)
		}
	}

	if cfg.Observability.Alerting.Slack.WebhookURL != "" {
		mgr, err := alerting.NewSlackManager(alerting.SlackConfig{
			WebhookURL: cfg.Observability.Alerting.Slack.WebhookURL,
			Channel:    cfg.Observability.Alerting.Slack.Channel,
		})
		if err == nil {
			managers = append(managers, mgr)
		}
	}

	if cfg.Observability.Alerting.Webhook.URL != "" {
		mgr, err := alerting.NewWebhookManager(alerting.WebhookConfig{
			URL: cfg.Observability.Alerting.Webhook.URL,
		})
		if err == nil {
			managers = append(managers, mgr)
		}
	}

	if len(managers) == 0 {
		return fmt.Errorf("no alert destinations configured")
	}

	result := &AlertTestResult{
		Destinations: make([]AlertDestinationResult, 0, len(managers)),
		Success:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Testing alert destinations...")

	for _, mgr := range managers {
		destResult := AlertDestinationResult{
			Name:    mgr.Name(),
			Success: true,
		}

		fmt.Printf("  Testing %s... ", mgr.Name())
		if err := mgr.Test(ctx); err != nil {
			destResult.Success = false
			destResult.Error = err.Error()
			result.Success = false
			fmt.Printf("FAILED: %s\n", err.Error())
		} else {
			fmt.Println("OK")
		}

		result.Destinations = append(result.Destinations, destResult)
	}

	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(result)
	}

	if result.Success {
		fmt.Println("\nAll alert destinations tested successfully!")
	} else {
		fmt.Println("\nSome alert destinations failed. Check the errors above.")
	}

	return nil
}

var alertsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show alert routing status",
	Long:  "Display the status of all configured alert destinations.",
	RunE:  runAlertsStatus,
}

func runAlertsStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	status := struct {
		Enabled      bool     `json:"enabled"`
		Destinations []string `json:"destinations"`
	}{
		Enabled:      cfg.Observability.Alerting.Enabled,
		Destinations: []string{},
	}

	if cfg.Observability.Alerting.PagerDuty.RoutingKey != "" {
		status.Destinations = append(status.Destinations, "pagerduty")
	}
	if cfg.Observability.Alerting.Opsgenie.APIKey != "" {
		status.Destinations = append(status.Destinations, "opsgenie")
	}
	if cfg.Observability.Alerting.Slack.WebhookURL != "" {
		status.Destinations = append(status.Destinations, "slack")
	}
	if cfg.Observability.Alerting.Webhook.URL != "" {
		status.Destinations = append(status.Destinations, "webhook")
	}

	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(status)
	}

	// Text output
	fmt.Println("Alerting Status")
	fmt.Println("===============")
	fmt.Printf("Enabled: %v\n", status.Enabled)
	fmt.Printf("Destinations: %v\n", status.Destinations)

	return nil
}

// ========================================
// Metrics Command
// ========================================

var obsMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "View current metrics",
	Long:  "Display currently tracked metrics and their values.",
	RunE:  runObsMetrics,
}

// MetricsReport represents the metrics output
type MetricsReport struct {
	TelemetryEnabled bool          `json:"telemetry_enabled"`
	Metrics          []MetricEntry `json:"metrics"`
}

// MetricEntry represents a single metric
type MetricEntry struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Help   string            `json:"help"`
	Labels map[string]string `json:"labels,omitempty"`
}

func runObsMetrics(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	report := &MetricsReport{
		TelemetryEnabled: cfg.Telemetry.Enabled,
		Metrics:          []MetricEntry{},
	}

	// List available metric types
	metricTypes := []struct {
		name string
		typ  string
		help string
	}{
		{"specular_command_executions_total", "counter", "Total command executions"},
		{"specular_command_duration_seconds", "histogram", "Command execution duration"},
		{"specular_provider_latency_seconds", "histogram", "Provider API latency"},
		{"specular_provider_tokens_total", "counter", "Total tokens used by provider"},
		{"specular_spec_validations_total", "counter", "Total spec validations"},
		{"specular_plan_generations_total", "counter", "Total plan generations"},
		{"specular_auto_workflows_total", "counter", "Total auto mode workflows"},
		{"specular_policy_evaluations_total", "counter", "Total policy evaluations"},
		{"specular_drift_detections_total", "counter", "Total drift detections"},
	}

	for _, m := range metricTypes {
		report.Metrics = append(report.Metrics, MetricEntry{
			Name: m.name,
			Type: m.typ,
			Help: m.help,
		})
	}

	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(report)
	}

	// Text output
	fmt.Println("Specular Metrics")
	fmt.Println("================")
	fmt.Printf("Telemetry Enabled: %v\n\n", report.TelemetryEnabled)

	fmt.Printf("%-45s %-12s %s\n", "Metric", "Type", "Description")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, m := range report.Metrics {
		fmt.Printf("%-45s %-12s %s\n", m.Name, m.Type, m.Help)
	}

	return nil
}
