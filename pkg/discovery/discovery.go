// Package discovery provides interfaces and types for command discovery
package discovery

import (
	"context"

	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

// Discoverer interface defines the contract for command discovery.
type Discoverer interface {
	// Discover finds commands based on the request parameters.
	Discover(ctx context.Context, req *types.CommandDiscoveryRequest) (*types.CommandDiscoveryResult, error)

	// ClearCache clears the discovery cache.
	ClearCache()
}

// DiscoveryBuilder helps build command discovery requests.
type DiscoveryBuilder struct {
	req *types.CommandDiscoveryRequest
}

// NewDiscoveryBuilder creates a new discovery builder.
func NewDiscoveryBuilder() *DiscoveryBuilder {
	return &DiscoveryBuilder{
		req: &types.CommandDiscoveryRequest{},
	}
}

// WithPattern sets the search pattern.
func (b *DiscoveryBuilder) WithPattern(pattern string) *DiscoveryBuilder {
	b.req.Pattern = pattern
	return b
}

// WithPaths sets additional paths to search.
func (b *DiscoveryBuilder) WithPaths(paths ...string) *DiscoveryBuilder {
	b.req.Paths = paths
	return b
}

// WithMaxResults sets the maximum number of results.
func (b *DiscoveryBuilder) WithMaxResults(max int) *DiscoveryBuilder {
	b.req.MaxResults = max
	return b
}

// WithDescriptions includes command descriptions in results.
func (b *DiscoveryBuilder) WithDescriptions(include bool) *DiscoveryBuilder {
	b.req.IncludeDesc = include
	return b
}

// Build returns the discovery request.
func (b *DiscoveryBuilder) Build() *types.CommandDiscoveryRequest {
	return b.req
}

// BuildAndDiscover builds the request and executes discovery.
func (b *DiscoveryBuilder) BuildAndDiscover(ctx context.Context, discoverer Discoverer) (*types.CommandDiscoveryResult, error) {
	return discoverer.Discover(ctx, b.req)
}

// Filter interface for filtering discovered commands.
type Filter interface {
	ShouldInclude(cmd types.CommandInfo) bool
}

// PatternFilter filters commands based on name patterns.
type PatternFilter struct {
	Patterns []string
}

// ShouldInclude implements the Filter interface.
func (f *PatternFilter) ShouldInclude(cmd types.CommandInfo) bool {
	if len(f.Patterns) == 0 {
		return true
	}

	for _, pattern := range f.Patterns {
		// Simple substring match
		if pattern == "*" || pattern == "" {
			return true
		}
		// Add more sophisticated pattern matching here if needed
		if cmd.Name == pattern {
			return true
		}
	}
	return false
}

// PathFilter filters commands based on their paths.
type PathFilter struct {
	AllowedPaths []string
}

// ShouldInclude implements the Filter interface.
func (f *PathFilter) ShouldInclude(cmd types.CommandInfo) bool {
	if len(f.AllowedPaths) == 0 {
		return true
	}

	for _, allowedPath := range f.AllowedPaths {
		if len(cmd.Path) >= len(allowedPath) && cmd.Path[:len(allowedPath)] == allowedPath {
			return true
		}
	}
	return false
}

// FilterChain chains multiple filters together.
type FilterChain struct {
	Filters []Filter
}

// ShouldInclude implements the Filter interface.
func (f *FilterChain) ShouldInclude(cmd types.CommandInfo) bool {
	for _, filter := range f.Filters {
		if !filter.ShouldInclude(cmd) {
			return false
		}
	}
	return true
}

// NewFilterChain creates a new filter chain.
func NewFilterChain(filters ...Filter) *FilterChain {
	return &FilterChain{
		Filters: filters,
	}
}