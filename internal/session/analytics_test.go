package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionAnalytics_TotalTokens(t *testing.T) {
	analytics := &SessionAnalytics{
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
		TotalTurns:       5,
		ToolCalls:        []ToolCall{{Name: "Read", Count: 3}},
		Duration:         time.Hour,
		StartTime:        time.Now().Add(-time.Hour),
	}

	assert.Equal(t, 1800, analytics.TotalTokens())
}

func TestSessionAnalytics_ContextPercent(t *testing.T) {
	analytics := &SessionAnalytics{
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
	}

	// 1800 tokens / 200000 limit * 100 = 0.9%
	assert.InDelta(t, 0.9, analytics.ContextPercent(200000), 0.01)
}

func TestSessionAnalytics_ContextPercent_DefaultLimit(t *testing.T) {
	analytics := &SessionAnalytics{
		InputTokens:  20000,
		OutputTokens: 0,
	}

	// 20000 tokens / 200000 default limit * 100 = 10%
	assert.InDelta(t, 10.0, analytics.ContextPercent(0), 0.01)
}

func TestSessionAnalytics_ZeroTokens(t *testing.T) {
	analytics := &SessionAnalytics{}

	assert.Equal(t, 0, analytics.TotalTokens())
	assert.InDelta(t, 0.0, analytics.ContextPercent(200000), 0.01)
}

func TestToolCall(t *testing.T) {
	tc := ToolCall{
		Name:  "Read",
		Count: 5,
	}

	assert.Equal(t, "Read", tc.Name)
	assert.Equal(t, 5, tc.Count)
}

func TestSubagentInfo(t *testing.T) {
	now := time.Now()
	sa := SubagentInfo{
		ID:        "subagent-123",
		StartTime: now,
		Turns:     10,
	}

	assert.Equal(t, "subagent-123", sa.ID)
	assert.Equal(t, now, sa.StartTime)
	assert.Equal(t, 10, sa.Turns)
}

func TestBillingBlock(t *testing.T) {
	start := time.Now()
	end := start.Add(5 * time.Hour)

	bb := BillingBlock{
		StartTime:  start,
		EndTime:    end,
		TokensUsed: 50000,
		IsActive:   true,
	}

	assert.Equal(t, start, bb.StartTime)
	assert.Equal(t, end, bb.EndTime)
	assert.Equal(t, 50000, bb.TokensUsed)
	assert.True(t, bb.IsActive)
}
