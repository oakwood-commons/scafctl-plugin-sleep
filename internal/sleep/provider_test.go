// Copyright 2025-2026 Oakwood Commons
// SPDX-License-Identifier: Apache-2.0

package sleep

import (
	"context"
	"testing"
	"time"

	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPlugin() *Plugin {
	return &Plugin{}
}

func TestGetProviders(t *testing.T) {
	providers, err := newPlugin().GetProviders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{ProviderName}, providers)
}

func TestGetProviderDescriptor(t *testing.T) {
	desc, err := newPlugin().GetProviderDescriptor(context.Background(), ProviderName)
	require.NoError(t, err)
	require.NotNil(t, desc)
	assert.Equal(t, "sleep", desc.Name)
	assert.Equal(t, "Sleep Provider", desc.DisplayName)
	assert.Equal(t, "v1", desc.APIVersion)
	assert.Equal(t, "1.0.0", desc.Version.String())
	assert.Len(t, desc.Capabilities, 5)
	assert.NotNil(t, desc.Schema)
	assert.Len(t, desc.OutputSchemas, 5)
	assert.Len(t, desc.Examples, 4)
}

func TestGetProviderDescriptor_UnknownProvider(t *testing.T) {
	_, err := newPlugin().GetProviderDescriptor(context.Background(), "unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestExecuteProvider_Success(t *testing.T) {
	p := newPlugin()

	start := time.Now()
	output, err := p.ExecuteProvider(context.Background(), ProviderName, map[string]any{
		"duration": "100ms",
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, output)

	data, ok := output.Data.(map[string]any)
	require.True(t, ok)
	assert.True(t, data["success"].(bool))
	assert.Equal(t, "100ms", data["duration"])
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
}

func TestExecuteProvider_InvalidDuration(t *testing.T) {
	output, err := newPlugin().ExecuteProvider(context.Background(), ProviderName, map[string]any{
		"duration": "invalid",
	})
	require.Error(t, err)
	require.Nil(t, output)
	assert.Contains(t, err.Error(), "invalid duration format")
}

func TestExecuteProvider_NegativeDuration(t *testing.T) {
	output, err := newPlugin().ExecuteProvider(context.Background(), ProviderName, map[string]any{
		"duration": "-5s",
	})
	require.Error(t, err)
	require.Nil(t, output)
	assert.Contains(t, err.Error(), "duration cannot be negative")
}

func TestExecuteProvider_MissingDuration(t *testing.T) {
	output, err := newPlugin().ExecuteProvider(context.Background(), ProviderName, map[string]any{})
	require.Error(t, err)
	require.Nil(t, output)
	assert.Contains(t, err.Error(), "duration is required")
}

func TestExecuteProvider_UnknownProvider(t *testing.T) {
	_, err := newPlugin().ExecuteProvider(context.Background(), "unknown", map[string]any{
		"duration": "1s",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestExecuteProvider_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	output, err := newPlugin().ExecuteProvider(ctx, ProviderName, map[string]any{
		"duration": "5s",
	})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Nil(t, output)
	assert.Contains(t, err.Error(), "sleep interrupted")
	assert.Less(t, elapsed, 5*time.Second)
}

func TestExecuteProvider_DryRun(t *testing.T) {
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	start := time.Now()
	output, err := newPlugin().ExecuteProvider(ctx, ProviderName, map[string]any{
		"duration": "5s",
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, output)

	data, ok := output.Data.(map[string]any)
	require.True(t, ok)
	assert.True(t, data["_dryRun"].(bool))
	assert.Equal(t, "5s", data["duration"])
	assert.Equal(t, "0s", data["elapsed"])
	assert.Less(t, elapsed, 100*time.Millisecond)
}

func TestExecuteProvider_DryRun_InvalidDuration(t *testing.T) {
	ctx := sdkprovider.WithDryRun(context.Background(), true)

	output, err := newPlugin().ExecuteProvider(ctx, ProviderName, map[string]any{
		"duration": "not-a-duration",
	})
	require.Error(t, err)
	require.Nil(t, output)
	assert.Contains(t, err.Error(), "invalid duration format")
}

func TestExecuteProvider_VariousDurations(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		minTime  time.Duration
	}{
		{"Milliseconds", "50ms", 50 * time.Millisecond},
		{"Complex", "150ms", 150 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			output, err := newPlugin().ExecuteProvider(context.Background(), ProviderName, map[string]any{
				"duration": tt.duration,
			})
			elapsed := time.Since(start)

			require.NoError(t, err)
			require.NotNil(t, output)

			data, ok := output.Data.(map[string]any)
			require.True(t, ok)
			assert.True(t, data["success"].(bool))
			assert.GreaterOrEqual(t, elapsed, tt.minTime)
		})
	}
}

func TestDescribeWhatIf(t *testing.T) {
	desc, err := newPlugin().DescribeWhatIf(context.Background(), ProviderName, map[string]any{
		"duration": "5s",
	})
	require.NoError(t, err)
	assert.Equal(t, "Would sleep for 5s", desc)
}

func TestDescribeWhatIf_NoDuration(t *testing.T) {
	desc, err := newPlugin().DescribeWhatIf(context.Background(), ProviderName, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "Would sleep for configured duration", desc)
}

func TestDescribeWhatIf_UnknownProvider(t *testing.T) {
	_, err := newPlugin().DescribeWhatIf(context.Background(), "unknown", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func BenchmarkExecuteProvider(b *testing.B) {
	p := newPlugin()
	ctx := sdkprovider.WithDryRun(context.Background(), true)
	input := map[string]any{"duration": "1s"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.ExecuteProvider(ctx, ProviderName, input)
	}
}

func BenchmarkGetProviderDescriptor(b *testing.B) {
	p := newPlugin()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.GetProviderDescriptor(ctx, ProviderName)
	}
}

func BenchmarkDescribeWhatIf(b *testing.B) {
	p := newPlugin()
	ctx := context.Background()
	input := map[string]any{"duration": "5s"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.DescribeWhatIf(ctx, ProviderName, input)
	}
}
