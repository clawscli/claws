package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

// Sentinel errors for action execution
var (
	ErrEmptyCommand        = errors.New("empty command")
	ErrInvalidResourceType = errors.New("invalid resource type")
)

// UnknownOperationError creates an error for unknown operations
func UnknownOperationError(operation string) error {
	return fmt.Errorf("unknown operation: %s", operation)
}

// InvalidResourceResult returns a standard result for invalid resource type
func InvalidResourceResult() ActionResult {
	return ActionResult{Success: false, Error: ErrInvalidResourceType}
}

// UnknownOperationResult returns a standard result for unknown operations
func UnknownOperationResult(operation string) ActionResult {
	return ActionResult{Success: false, Error: UnknownOperationError(operation)}
}

// ActionType represents the type of action
type ActionType string

const (
	ActionTypeExec ActionType = "exec" // Execute external command
	ActionTypeAPI  ActionType = "api"  // Call AWS API
	ActionTypeView ActionType = "view" // Navigate to another view
)

// Action defines an action that can be performed on a resource
type Action struct {
	Name      string
	Shortcut  string
	Type      ActionType
	Command   string            // For exec type
	Operation string            // For api type
	Target    string            // For view type
	Confirm   bool              // Require confirmation
	Dangerous bool              // Show warning
	Requires  []string          // Required dependencies
	Vars      map[string]string // Variable mappings
}

// ActionResult represents the result of an action
type ActionResult struct {
	Success bool
	Message string
	Error   error
}

// ExecutorFunc is a function that executes an action on a resource
type ExecutorFunc func(ctx context.Context, action Action, resource dao.Resource) ActionResult

// Registry holds actions for resources
type Registry struct {
	mu        sync.RWMutex
	actions   map[string][]Action     // key: service/resource
	executors map[string]ExecutorFunc // key: service/resource
}

// NewRegistry creates a new action registry
func NewRegistry() *Registry {
	return &Registry{
		actions:   make(map[string][]Action),
		executors: make(map[string]ExecutorFunc),
	}
}

// Register registers actions for a resource type
func (r *Registry) Register(service, resource string, actions []Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s/%s", service, resource)
	r.actions[key] = actions
}

// Get returns actions for a resource type
func (r *Registry) Get(service, resource string) []Action {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := fmt.Sprintf("%s/%s", service, resource)
	return r.actions[key]
}

// RegisterExecutor registers an executor for a resource type
func (r *Registry) RegisterExecutor(service, resource string, executor ExecutorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s/%s", service, resource)
	r.executors[key] = executor
}

// GetExecutor returns the executor for a resource type
func (r *Registry) GetExecutor(service, resource string) ExecutorFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := fmt.Sprintf("%s/%s", service, resource)
	return r.executors[key]
}

// RegisterExecutor is a convenience function to register with the global registry
func RegisterExecutor(service, resource string, executor ExecutorFunc) {
	Global.RegisterExecutor(service, resource, executor)
}

// ExecuteWithDAO executes an action with service/resource context for executor lookup
func ExecuteWithDAO(ctx context.Context, action Action, resource dao.Resource, service, resourceType string) ActionResult {
	log.Info("executing action", "action", action.Name, "type", action.Type, "service", service, "resourceType", resourceType, "resourceID", resource.GetID())

	var result ActionResult
	switch action.Type {
	case ActionTypeExec:
		result = executeExec(ctx, action, resource)
	case ActionTypeAPI:
		if executor := Global.GetExecutor(service, resourceType); executor != nil {
			result = executor(ctx, action, resource)
		} else {
			result = ActionResult{Success: false, Error: fmt.Errorf("no executor registered for %s/%s", service, resourceType)}
		}
	default:
		result = ActionResult{Success: false, Error: fmt.Errorf("unknown action type: %s", action.Type)}
	}

	if result.Success {
		log.Info("action completed", "action", action.Name, "success", true)
	} else {
		log.Error("action failed", "action", action.Name, "error", result.Error)
	}

	return result
}

func executeExec(ctx context.Context, action Action, resource dao.Resource) ActionResult {
	cmd, err := ExpandVariables(action.Command, resource)
	if err != nil {
		return ActionResult{Success: false, Error: err}
	}
	if cmd == "" {
		return ActionResult{Success: false, Error: ErrEmptyCommand}
	}

	// Execute command through shell to properly handle quoted arguments,
	// pipes, redirections, and other shell features
	execCmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmd)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	err = execCmd.Run()
	if err != nil {
		return ActionResult{Success: false, Error: err}
	}

	return ActionResult{Success: true, Message: "Command executed successfully"}
}

// Optional interfaces for variable expansion in action commands.
// Resources can implement these to provide additional variables.
type (
	// PrivateIPProvider provides ${PRIVATE_IP} variable (EC2 instances)
	PrivateIPProvider interface {
		PrivateIP() string
	}

	// ClusterArnProvider provides ${CLUSTER} variable (ECS services/tasks)
	ClusterArnProvider interface {
		ClusterArn() string
	}

	// ContainerNameProvider provides ${CONTAINER} variable (ECS tasks)
	ContainerNameProvider interface {
		FirstContainerName() string
	}

	// LogGroupNameProvider provides ${LOG_GROUP} variable (CloudWatch log streams)
	LogGroupNameProvider interface {
		LogGroupName() string
	}
)

// ErrUnsafeValue is returned when a variable value contains shell metacharacters
var ErrUnsafeValue = errors.New("variable value contains unsafe characters")

// ExpandVariables replaces variables in command strings with resource values.
// Standard variables: ${ID}, ${NAME}, ${ARN}, ${INSTANCE_ID}, ${BUCKET}
// Optional variables (if resource implements the interface):
//   - ${PRIVATE_IP} - PrivateIPProvider
//   - ${CLUSTER} - ClusterArnProvider
//   - ${CONTAINER} - ContainerNameProvider
//   - ${LOG_GROUP} - LogGroupNameProvider
//
// Returns an error if any value contains shell metacharacters.
func ExpandVariables(cmd string, resource dao.Resource) (string, error) {
	replacements := map[string]string{
		"${ID}":          resource.GetID(),
		"${NAME}":        resource.GetName(),
		"${ARN}":         resource.GetARN(),
		"${INSTANCE_ID}": resource.GetID(),
		"${BUCKET}":      resource.GetID(),
	}

	// Optional variables from interface implementations
	if p, ok := resource.(PrivateIPProvider); ok {
		replacements["${PRIVATE_IP}"] = p.PrivateIP()
	}
	if p, ok := resource.(ClusterArnProvider); ok {
		replacements["${CLUSTER}"] = p.ClusterArn()
	}
	if p, ok := resource.(ContainerNameProvider); ok {
		replacements["${CONTAINER}"] = p.FirstContainerName()
	}
	if p, ok := resource.(LogGroupNameProvider); ok {
		replacements["${LOG_GROUP}"] = p.LogGroupName()
	}

	// Check for unsafe characters in values that will be substituted
	for k, v := range replacements {
		if strings.Contains(cmd, k) && containsShellMetachar(v) {
			return "", fmt.Errorf("%w: %s contains shell metacharacters", ErrUnsafeValue, k)
		}
	}

	result := cmd
	for k, v := range replacements {
		result = strings.ReplaceAll(result, k, v)
	}
	return result, nil
}

// containsShellMetachar checks if a string contains shell metacharacters
// that could be used for command injection.
func containsShellMetachar(s string) bool {
	// Check for characters that have special meaning in shell
	for _, c := range s {
		switch c {
		case ';', '|', '&', '$', '`', '(', ')', '{', '}', '<', '>', '\n', '\r':
			return true
		}
	}
	return false
}

// Global is the default global action registry
var Global = NewRegistry()
