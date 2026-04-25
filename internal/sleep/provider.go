// Copyright 2025-2026 Oakwood Commons
// SPDX-License-Identifier: Apache-2.0

// Package sleep implements the sleep provider plugin.
package sleep

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/google/jsonschema-go/jsonschema"
	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	sdkhelper "github.com/oakwood-commons/scafctl-plugin-sdk/provider/schemahelper"
)

const (
	// ProviderName is the unique identifier for this provider.
	ProviderName = "sleep"

	// Version is the provider version.
	Version = "1.0.0"
)

// Plugin implements the scafctl ProviderPlugin interface.
type Plugin struct{}

// GetProviders returns the list of providers exposed by this plugin.
//
//nolint:revive // ctx required by interface
func (p *Plugin) GetProviders(_ context.Context) ([]string, error) {
	return []string{ProviderName}, nil
}

// GetProviderDescriptor returns the descriptor for the named provider.
//
//nolint:revive // ctx required by interface
func (p *Plugin) GetProviderDescriptor(_ context.Context, providerName string) (*sdkprovider.Descriptor, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	commonOutputSchema := sdkhelper.ObjectSchema(nil, map[string]*jsonschema.Schema{
		"duration": sdkhelper.StringProp("The duration that was slept"),
		"elapsed":  sdkhelper.StringProp("The actual elapsed time"),
	})

	return &sdkprovider.Descriptor{
		Name:        ProviderName,
		DisplayName: "Sleep Provider",
		Description: "Provides sleep/delay functionality for workflow control. Useful for rate limiting, waiting for external systems, or pacing workflow execution.",
		APIVersion:  "v1",
		Version:     semver.MustParse(Version),
		Category:    "Utility",
		Tags:        []string{"sleep", "delay", "wait", "workflow", "utility"},
		Capabilities: []sdkprovider.Capability{
			sdkprovider.CapabilityFrom,
			sdkprovider.CapabilityTransform,
			sdkprovider.CapabilityValidation,
			sdkprovider.CapabilityAuthentication,
			sdkprovider.CapabilityAction,
		},
		Schema: sdkhelper.ObjectSchema(
			[]string{"duration"},
			map[string]*jsonschema.Schema{
				"duration": sdkhelper.StringProp(
					"Duration to sleep. Accepts Go duration format (e.g., '1s', '500ms', '2m', '1h30m'). Valid time units are 'ns', 'us' (or 'us'), 'ms', 's', 'm', 'h'.",
					sdkhelper.WithExample("5s"),
					sdkhelper.WithMaxLength(50),
					sdkhelper.WithPattern(`^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$`),
				),
			},
		),
		OutputSchemas: map[sdkprovider.Capability]*jsonschema.Schema{
			sdkprovider.CapabilityFrom:      commonOutputSchema,
			sdkprovider.CapabilityTransform: commonOutputSchema,
			sdkprovider.CapabilityValidation: sdkhelper.ObjectSchema(nil, map[string]*jsonschema.Schema{
				"valid":    sdkhelper.BoolProp("Whether the sleep operation completed successfully (always true if no error)"),
				"errors":   sdkhelper.ArrayProp("Validation errors (empty if valid)"),
				"duration": sdkhelper.StringProp("The duration that was slept"),
				"elapsed":  sdkhelper.StringProp("The actual elapsed time"),
			}),
			sdkprovider.CapabilityAuthentication: sdkhelper.ObjectSchema(nil, map[string]*jsonschema.Schema{
				"authenticated": sdkhelper.BoolProp("Whether authentication succeeded (always true for sleep)"),
				"token":         sdkhelper.StringProp("The authentication token (empty for sleep provider)"),
				"duration":      sdkhelper.StringProp("The duration that was slept"),
				"elapsed":       sdkhelper.StringProp("The actual elapsed time"),
			}),
			sdkprovider.CapabilityAction: sdkhelper.ObjectSchema(nil, map[string]*jsonschema.Schema{
				"success":  sdkhelper.BoolProp("Whether the sleep operation completed successfully"),
				"duration": sdkhelper.StringProp("The duration that was slept"),
				"elapsed":  sdkhelper.StringProp("The actual elapsed time"),
			}),
		},
		Examples: []sdkprovider.Example{
			{
				Name:        "Short delay",
				Description: "Pause workflow execution for 5 seconds",
				YAML: `name: wait-5-seconds
provider: sleep
inputs:
  duration: "5s"`,
			},
			{
				Name:        "Millisecond precision delay",
				Description: "Pause for 500 milliseconds for precise timing control",
				YAML: `name: half-second-delay
provider: sleep
inputs:
  duration: "500ms"`,
			},
			{
				Name:        "Rate limiting delay",
				Description: "Wait 2 minutes between API calls to respect rate limits",
				YAML: `name: rate-limit-pause
provider: sleep
inputs:
  duration: "2m"`,
			},
			{
				Name:        "Combined duration units",
				Description: "Use multiple time units for complex durations (1 hour 30 minutes)",
				YAML: `name: long-wait
provider: sleep
inputs:
  duration: "1h30m"`,
			},
		},
	}, nil
}

// ExecuteProvider performs the sleep operation.
func (p *Plugin) ExecuteProvider(ctx context.Context, providerName string, inputs map[string]any) (*sdkprovider.Output, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	lgr := logr.FromContextOrDiscard(ctx)

	// Check for dry-run mode
	if sdkprovider.DryRunFromContext(ctx) {
		return p.executeDryRun(inputs)
	}

	// Validate and parse duration
	durationStr, ok := inputs["duration"].(string)
	if !ok || durationStr == "" {
		return nil, fmt.Errorf("%s: duration is required and must be a string", ProviderName)
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid duration format: %w (expected format like '1s', '500ms', '2m')", ProviderName, err)
	}

	if duration < 0 {
		return nil, fmt.Errorf("%s: duration cannot be negative: %s", ProviderName, durationStr)
	}

	lgr.V(1).Info("Starting sleep", "duration", durationStr)

	// Perform the sleep with context cancellation support
	start := time.Now()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		elapsed := time.Since(start)
		lgr.V(1).Info("Sleep completed", "duration", durationStr, "elapsed", elapsed)

		return &sdkprovider.Output{
			Data: map[string]any{
				"success":  true,
				"duration": durationStr,
				"elapsed":  elapsed.String(),
			},
		}, nil

	case <-ctx.Done():
		elapsed := time.Since(start)
		lgr.V(1).Info("Sleep interrupted by context cancellation", "duration", durationStr, "elapsed", elapsed)
		return nil, fmt.Errorf("%s: sleep interrupted: %w", ProviderName, ctx.Err())
	}
}

// executeDryRun handles dry-run mode without actually sleeping.
func (p *Plugin) executeDryRun(inputs map[string]any) (*sdkprovider.Output, error) {
	durationStr, _ := inputs["duration"].(string)

	// Validate duration format even in dry-run
	if durationStr != "" {
		if _, err := time.ParseDuration(durationStr); err != nil {
			return nil, fmt.Errorf("%s: invalid duration format: %w (expected format like '1s', '500ms', '2m')", ProviderName, err)
		}
	}

	return &sdkprovider.Output{
		Data: map[string]any{
			"success":  true,
			"duration": durationStr,
			"elapsed":  "0s",
			"_dryRun":  true,
			"_message": fmt.Sprintf("Would sleep for %s", durationStr),
		},
	}, nil
}

// DescribeWhatIf returns a description of what the provider would do.
//
//nolint:revive // ctx required by interface
func (p *Plugin) DescribeWhatIf(_ context.Context, providerName string, inputs map[string]any) (string, error) {
	if providerName != ProviderName {
		return "", fmt.Errorf("unknown provider: %s", providerName)
	}

	duration, _ := inputs["duration"].(string)
	if duration != "" {
		return fmt.Sprintf("Would sleep for %s", duration), nil
	}
	return "Would sleep for configured duration", nil
}

// ConfigureProvider stores host-side configuration. The sleep plugin does not
// require any configuration, so this is a no-op.
//
//nolint:revive // ctx and cfg required by interface
func (p *Plugin) ConfigureProvider(_ context.Context, _ string, _ sdkplugin.ProviderConfig) error {
	return nil
}

// ExecuteProviderStream is not supported by the sleep plugin.
//
//nolint:revive // all params required by interface
func (p *Plugin) ExecuteProviderStream(_ context.Context, _ string, _ map[string]any, _ func(sdkplugin.StreamChunk)) error {
	return sdkplugin.ErrStreamingNotSupported
}

// ExtractDependencies returns resolver keys this input depends on.
//
//nolint:revive // all params required by interface
func (p *Plugin) ExtractDependencies(_ context.Context, _ string, _ map[string]any) ([]string, error) {
	return nil, nil
}

// StopProvider performs cleanup for the named provider.
//
//nolint:revive // all params required by interface
func (p *Plugin) StopProvider(_ context.Context, _ string) error {
	return nil
}
