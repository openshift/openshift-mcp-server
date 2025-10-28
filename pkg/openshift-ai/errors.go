package openshiftai

import (
	"fmt"
)

// Common error constructors using simple Go error patterns
func NotFoundError(resource, name string) error {
	return fmt.Errorf("%s '%s' not found", resource, name)
}

func AlreadyExistsError(resource, name string) error {
	return fmt.Errorf("%s '%s' already exists", resource, name)
}

func InvalidArgumentError(message string) error {
	return fmt.Errorf("invalid argument: %s", message)
}

func PermissionDeniedError(action, resource string) error {
	return fmt.Errorf("permission denied for %s on %s", action, resource)
}

func UnavailableError(service string) error {
	return fmt.Errorf("service '%s' is unavailable", service)
}

func TimeoutError(operation string) error {
	return fmt.Errorf("operation '%s' timed out", operation)
}

func InternalError(message string, cause error) error {
	return fmt.Errorf("%s: %w", message, cause)
}
