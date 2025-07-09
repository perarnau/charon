package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// PrometheusResponse represents the structure of Prometheus query response
type PrometheusResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

func (c *CLI) executeMetrics(args []string) error {
	fmt.Println("📊 Metrics command executed")

	if len(args) == 0 {
		fmt.Println("   Usage: metrics <prometheus-url> [options]")
		fmt.Println("   Options:")
		fmt.Println("     --start <timestamp>     Start time (RFC3339 or Unix timestamp)")
		fmt.Println("     --end <timestamp>       End time (RFC3339 or Unix timestamp)")
		fmt.Println("     --step <duration>       Step duration (e.g., 30s, 1m, 5m)")
		fmt.Println("     --query <query>         PromQL query to execute (can be used multiple times)")
		fmt.Println("     --queries-file <file>   File containing list of PromQL queries (one per line)")
		fmt.Println("     --output <file>         Output file path (default: metrics.json)")
		fmt.Println("     --list-metrics          List all available metrics from Prometheus")
		fmt.Println("")
		fmt.Println("   Examples:")
		fmt.Println("     metrics http://localhost:9090 --list-metrics")
		fmt.Println("     metrics http://localhost:9090")
		fmt.Println("     metrics http://localhost:9090 --query 'up' --query 'prometheus_tsdb_head_series'")
		fmt.Println("     metrics http://localhost:9090 --queries-file my_queries.txt --output metrics.json")
		fmt.Println("     metrics http://localhost:9090 --start 2024-01-01T00:00:00Z --end 2024-01-01T01:00:00Z")
		fmt.Println("     metrics http://localhost:9090 --step 1m --query 'cpu_usage_percent'")
		fmt.Println("")
		fmt.Println("   Queries file format (one query per line):")
		fmt.Println("     up")
		fmt.Println("     prometheus_tsdb_head_series")
		fmt.Println("     rate(http_requests_total[5m])")
		fmt.Println("")
		fmt.Println("   Port-forward setup for Kubernetes:")
		fmt.Println("     kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090")
		return fmt.Errorf("missing prometheus URL argument")
	}

	prometheusURL := args[0]
	
	// Parse command line options
	options, err := c.parseMetricsOptions(args[1:])
	if err != nil {
		return fmt.Errorf("error parsing options: %v", err)
	}

	fmt.Printf("   Prometheus URL: %s\n", prometheusURL)
	fmt.Printf("   Output file: %s\n", options.outputFile)
	
	if len(options.queries) > 0 {
		fmt.Printf("   Queries (%d):\n", len(options.queries))
		for i, query := range options.queries {
			fmt.Printf("     %d. %s\n", i+1, query)
		}
	}
	
	if options.queriesFile != "" {
		fmt.Printf("   Queries file: %s\n", options.queriesFile)
	}
	
	if !options.startTime.IsZero() {
		fmt.Printf("   Start time: %s\n", options.startTime.Format(time.RFC3339))
	}
	
	if !options.endTime.IsZero() {
		fmt.Printf("   End time: %s\n", options.endTime.Format(time.RFC3339))
	}
	
	if options.step != "" {
		fmt.Printf("   Step: %s\n", options.step)
	}

	// Validate Prometheus URL
	if !strings.HasPrefix(prometheusURL, "http://") && !strings.HasPrefix(prometheusURL, "https://") {
		return fmt.Errorf("prometheus URL must start with http:// or https://")
	}

	// Test connection to Prometheus
	fmt.Println("   Status: Testing connection to Prometheus...")
	if err := c.testPrometheusConnection(prometheusURL); err != nil {
		return fmt.Errorf("failed to connect to Prometheus: %v", err)
	}
	fmt.Println("   ✅ Successfully connected to Prometheus")

	// Check if user wants to list metrics
	if options.listMetrics {
		fmt.Println("   Status: Fetching all available metrics...")
		if err := c.listPrometheusMetrics(prometheusURL); err != nil {
			return fmt.Errorf("failed to list metrics: %v", err)
		}
		return nil
	}

	// Collect metrics
	fmt.Println("   Status: Collecting metrics...")
	
	var metricsData []PrometheusResponse
	var queriesToExecute []string
	
	// Determine which queries to execute
	if len(options.queries) > 0 {
		queriesToExecute = options.queries
	} else if options.queriesFile != "" {
		// Read queries from file
		fileQueries, err := c.readQueriesFromFile(options.queriesFile)
		if err != nil {
			return fmt.Errorf("failed to read queries file: %v", err)
		}
		queriesToExecute = fileQueries
	} else {
		// If no specific queries, use default common metrics
		queriesToExecute = []string{
			"up",
			"prometheus_tsdb_head_samples_appended_total",
			"prometheus_tsdb_head_series",
		}
	}
	
	fmt.Printf("   Status: Executing %d queries...\n", len(queriesToExecute))
	
	// Execute all queries
	for i, query := range queriesToExecute {
		fmt.Printf("   [%d/%d] Querying: %s\n", i+1, len(queriesToExecute), query)
		
		// Create a copy of options with the current query
		queryOptions := *options
		queryOptions.query = query
		
		data, err := c.queryPrometheus(prometheusURL, &queryOptions)
		if err != nil {
			fmt.Printf("   ⚠️  Warning: Failed to query '%s': %v\n", query, err)
			continue
		}
		metricsData = append(metricsData, *data)
	}

	// Save metrics to file
	fmt.Printf("   Status: Saving metrics to %s...\n", options.outputFile)
	if err := c.saveMetricsToFile(metricsData, options.outputFile, queriesToExecute); err != nil {
		return fmt.Errorf("failed to save metrics: %v", err)
	}

	fmt.Printf("   ✅ Successfully saved %d metric queries to %s\n", len(metricsData), options.outputFile)
	return nil
}

// MetricsOptions holds the parsed command line options for metrics command
type MetricsOptions struct {
	startTime    time.Time
	endTime      time.Time
	step         string
	query        string        // Used internally for individual queries
	queries      []string      // Multiple queries from --query flags
	queriesFile  string        // File containing queries
	outputFile   string
	listMetrics  bool
}

// parseMetricsOptions parses command line arguments for the metrics command
func (c *CLI) parseMetricsOptions(args []string) (*MetricsOptions, error) {
	options := &MetricsOptions{
		outputFile: "metrics.json",
		step:       "30s",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--start":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--start requires a timestamp value")
			}
			i++
			startTime, err := parseTimestamp(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid start time: %v", err)
			}
			options.startTime = startTime

		case "--end":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--end requires a timestamp value")
			}
			i++
			endTime, err := parseTimestamp(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid end time: %v", err)
			}
			options.endTime = endTime

		case "--step":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--step requires a duration value")
			}
			i++
			options.step = args[i]

		case "--query":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--query requires a PromQL query")
			}
			i++
			options.queries = append(options.queries, args[i])

		case "--queries-file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--queries-file requires a file path")
			}
			i++
			options.queriesFile = args[i]

		case "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--output requires a file path")
			}
			i++
			options.outputFile = args[i]

		case "--list-metrics":
			options.listMetrics = true

		default:
			return nil, fmt.Errorf("unknown option: %s", args[i])
		}
	}

	// Validate that queries and queries-file are not both specified
	if len(options.queries) > 0 && options.queriesFile != "" {
		return nil, fmt.Errorf("cannot specify both --query and --queries-file options")
	}

	// Set default time range if not specified
	// Only set default time range if user hasn't specified any time-related options
	if options.startTime.IsZero() && options.endTime.IsZero() && len(options.queries) == 0 && options.queriesFile == "" {
		// For default queries, use range queries with last hour
		options.endTime = time.Now()
		options.startTime = options.endTime.Add(-1 * time.Hour)
	} else if !options.startTime.IsZero() && options.endTime.IsZero() {
		options.endTime = time.Now()
	} else if options.startTime.IsZero() && !options.endTime.IsZero() {
		options.startTime = options.endTime.Add(-1 * time.Hour)
	}
	// If both startTime and endTime are zero, do instant queries

	return options, nil
}

// parseTimestamp parses various timestamp formats
func parseTimestamp(ts string) (time.Time, error) {
	// Try parsing as Unix timestamp first
	if unix, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.Unix(unix, 0), nil
	}

	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, nil
	}

	// Try parsing as common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", ts)
}

// testPrometheusConnection tests if Prometheus is accessible
func (c *CLI) testPrometheusConnection(prometheusURL string) error {
	// Test the /api/v1/query endpoint with a simple query
	testURL := fmt.Sprintf("%s/api/v1/query?query=up", prometheusURL)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(testURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Prometheus returned status %d", resp.StatusCode)
	}

	return nil
}

// queryPrometheus executes a query against Prometheus
func (c *CLI) queryPrometheus(prometheusURL string, options *MetricsOptions) (*PrometheusResponse, error) {
	var apiURL string

	// Build the appropriate API endpoint
	if !options.startTime.IsZero() && !options.endTime.IsZero() {
		// Range query
		apiURL = fmt.Sprintf("%s/api/v1/query_range", prometheusURL)
	} else {
		// Instant query
		apiURL = fmt.Sprintf("%s/api/v1/query", prometheusURL)
	}

	// Build query parameters
	params := url.Values{}
	params.Add("query", options.query)

	if !options.startTime.IsZero() && !options.endTime.IsZero() {
		params.Add("start", strconv.FormatInt(options.startTime.Unix(), 10))
		params.Add("end", strconv.FormatInt(options.endTime.Unix(), 10))
		params.Add("step", options.step)
	}

	// Make the request
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var promResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode Prometheus response: %v", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("Prometheus query failed with status: %s", promResp.Status)
	}

	return &promResp, nil
}

// saveMetricsToFile saves the collected metrics to a JSON file
func (c *CLI) saveMetricsToFile(metricsData []PrometheusResponse, outputFile string, queries []string) error {
	// Create a structured output with query information
	results := make([]map[string]interface{}, len(metricsData))
	
	for i, data := range metricsData {
		var queryName string
		if i < len(queries) {
			queryName = queries[i]
		} else {
			queryName = fmt.Sprintf("query_%d", i+1)
		}
		
		results[i] = map[string]interface{}{
			"query": queryName,
			"data":  data,
		}
	}
	
	output := map[string]interface{}{
		"timestamp":     time.Now().Format(time.RFC3339),
		"total_queries": len(metricsData),
		"results":       results,
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// listPrometheusMetrics fetches and displays all available metrics from Prometheus
func (c *CLI) listPrometheusMetrics(prometheusURL string) error {
	// Use the /api/v1/label/__name__/values endpoint to get all metric names
	apiURL := fmt.Sprintf("%s/api/v1/label/__name__/values", prometheusURL)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Prometheus API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var metricNamesResponse struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&metricNamesResponse); err != nil {
		return fmt.Errorf("failed to decode Prometheus response: %v", err)
	}

	if metricNamesResponse.Status != "success" {
		return fmt.Errorf("Prometheus query failed with status: %s", metricNamesResponse.Status)
	}

	// Display the metrics
	fmt.Printf("   📋 Found %d available metrics:\n\n", len(metricNamesResponse.Data))
	
	// Group metrics by common prefixes for better organization
	metricGroups := make(map[string][]string)
	ungrouped := []string{}
	
	for _, metric := range metricNamesResponse.Data {
		// Try to find a common prefix (before first underscore)
		parts := strings.SplitN(metric, "_", 2)
		if len(parts) > 1 {
			prefix := parts[0]
			metricGroups[prefix] = append(metricGroups[prefix], metric)
		} else {
			ungrouped = append(ungrouped, metric)
		}
	}

	// Display grouped metrics
	fmt.Println("   📊 Metrics by category:")
	fmt.Println("   " + strings.Repeat("=", 50))
	
	for prefix, metrics := range metricGroups {
		if len(metrics) > 1 { // Only show as group if more than 1 metric
			fmt.Printf("   🏷️  %s (%d metrics):\n", strings.ToUpper(prefix), len(metrics))
			for _, metric := range metrics {
				fmt.Printf("      • %s\n", metric)
			}
			fmt.Println()
		} else {
			ungrouped = append(ungrouped, metrics[0])
		}
	}
	
	if len(ungrouped) > 0 {
		fmt.Printf("   🏷️  OTHER (%d metrics):\n", len(ungrouped))
		for _, metric := range ungrouped {
			fmt.Printf("      • %s\n", metric)
		}
		fmt.Println()
	}

	fmt.Println("   💡 Usage examples:")
	fmt.Println("      charonctl metrics http://localhost:9090 --query 'up'")
	fmt.Println("      charonctl metrics http://localhost:9090 --query 'prometheus_tsdb_head_series'")
	if len(metricNamesResponse.Data) > 0 {
		fmt.Printf("      charonctl metrics http://localhost:9090 --query '%s'\n", metricNamesResponse.Data[0])
	}

	return nil
}

// readQueriesFromFile reads PromQL queries from a file, one per line
func (c *CLI) readQueriesFromFile(filename string) ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("queries file '%s' does not exist", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open queries file: %v", err)
	}
	defer file.Close()

	var queries []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments (lines starting with #)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		queries = append(queries, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading queries file at line %d: %v", lineNum, err)
	}

	if len(queries) == 0 {
		return nil, fmt.Errorf("no valid queries found in file '%s'", filename)
	}

	fmt.Printf("   📄 Loaded %d queries from %s\n", len(queries), filename)
	return queries, nil
}
