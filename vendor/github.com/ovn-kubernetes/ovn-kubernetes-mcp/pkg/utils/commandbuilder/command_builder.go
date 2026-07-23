package commandbuilder

import "slices"

// CommandBuilder helps build commands with a fluent interface
type CommandBuilder struct {
	args []string
}

// NewCommand creates a new command builder with the base command
func NewCommand(baseCmd ...string) *CommandBuilder {
	return &CommandBuilder{args: slices.Clone(baseCmd)}
}

// Add adds arguments to the command
func (cb *CommandBuilder) Add(args ...string) *CommandBuilder {
	cb.args = append(cb.args, args...)
	return cb
}

// AddIf adds arguments to the command only if the condition is true
func (cb *CommandBuilder) AddIf(condition bool, args ...string) *CommandBuilder {
	if condition {
		cb.args = append(cb.args, args...)
	}
	return cb
}

// AddIfNotEmpty adds arguments to the command only if the value is not empty
func (cb *CommandBuilder) AddIfNotEmpty(value string, args ...string) *CommandBuilder {
	if value != "" {
		cb.args = append(cb.args, args...)
	}
	return cb
}

// Build returns the final command slice
func (cb *CommandBuilder) Build() []string {
	return slices.Clone(cb.args)
}
