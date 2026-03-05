package logging

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesProcessed tracks the total number of messages handled by agents.
	MessagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wonderpus_messages_total",
		Help: "The total number of processed messages",
	}, []string{"channel", "provider"})

	// ToolExecutionTime tracks how long tools take to execute.
	ToolExecutionTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wonderpus_tool_execution_seconds",
		Help:    "Execution time of tools in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool"})

	// AgentErrors tracks the number of errors encountered by agents.
	AgentErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wonderpus_agent_errors_total",
		Help: "The total number of agent errors",
	}, []string{"type"})

	// TokenUsage tracks total tokens used.
	TokenUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wonderpus_tokens_total",
		Help: "Total tokens consumed",
	}, []string{"model", "type"}) // type: input, output

	// ProviderCost tracks total cost in USD.
	ProviderCost = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wonderpus_cost_usd_total",
		Help: "Total estimated provider cost in USD",
	}, []string{"model", "session"})

	// ProviderLatency tracks LLM response times.
	ProviderLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wonderpus_provider_latency_seconds",
		Help:    "LLM provider response time in seconds",
		Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60},
	}, []string{"provider", "model"})
)
