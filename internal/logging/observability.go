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
)
