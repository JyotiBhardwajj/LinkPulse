package health

import "context"

// Checker defines a contract for executing health status checks on a dependency.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
	IsCritical() bool
}
