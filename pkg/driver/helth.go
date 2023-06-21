package driver

import "context"

// HealCheck is the interface that must be implemented to be compatible with `HealthChecker`.
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

type HealthChecker struct {
	checks []HealthCheck
}

func NewHealthChecker(checks ...HealthCheck) *HealthChecker {
	return &HealthChecker{
		checks: checks,
	}
}

type doHealthChecker struct {
	account godo.AccountService
}
