package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// Sentinel errors for action execution
var (
	ErrEmptyCommand        = errors.New("empty command")
	ErrEmptyOperation      = errors.New("API action has no Operation defined")
	ErrInvalidResourceType = errors.New("invalid resource type")
	ErrReadOnlyDenied      = errors.New("action denied in read-only mode")
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

type ActionType string

const (
	ActionTypeExec ActionType = "exec"
	ActionTypeAPI  ActionType = "api"
)

type ConfirmLevel int

const (
	ConfirmNone ConfirmLevel = iota
	ConfirmSimple
	ConfirmDangerous
)

const (
	ActionNameSSOLogin = "SSO Login"
	ActionNameLogin    = "Login"
)

type Action struct {
	Name      string
	Shortcut  string
	Type      ActionType
	Command   string
	Operation string
	Confirm   ConfirmLevel

	// SkipAWSEnv skips AWS env injection for exec commands.
	// Use for commands that need to access ~/.aws files directly (e.g., aws sso login).
	SkipAWSEnv bool

	// Filter returns true if this action should be shown for the given resource.
	// If nil, the action is always shown.
	Filter func(resource dao.Resource) bool

	// PostExecFollowUp generates a tea.Msg after successful exec completion.
	// Called by ActionMenu when an exec action returns success.
	// If nil, no follow-up message is sent.
	// Example: profile switch after SSO login returns msg.ProfilesChangedMsg.
	PostExecFollowUp func(resource dao.Resource) any

	// ConfirmToken returns the string the user must type to confirm dangerous actions.
	// If nil, defaults to resource.GetID().
	// Use when the action operates on a different identifier (e.g., Name vs ARN).
	ConfirmToken func(resource dao.Resource) string
}

// ActionResult represents the result of an action
type ActionResult struct {
	Success     bool
	Message     string
	Error       error
	ErrorKind   apperrors.Kind // Classification of the error (Auth, Throttling, NotFound, etc.)
	FollowUpMsg any            // Optional tea.Msg to send after action completes
}

// FailResult creates a failed ActionResult with automatic error classification.
func FailResult(err error) ActionResult {
	return ActionResult{
		Success:   false,
		Error:     err,
		ErrorKind: apperrors.Classify(err),
	}
}

// FailResultf creates a failed ActionResult with formatted message and classification.
func FailResultf(err error, format string, args ...any) ActionResult {
	wrapped := apperrors.Wrapf(err, format, args...)
	return ActionResult{
		Success:   false,
		Error:     wrapped,
		ErrorKind: apperrors.Classify(err),
	}
}

// SuccessResult creates a successful ActionResult with a message.
func SuccessResult(message string) ActionResult {
	return ActionResult{
		Success: true,
		Message: message,
	}
}

// SuccessResultWithFollowUp creates a successful ActionResult with a follow-up message.
func SuccessResultWithFollowUp(message string, followUp any) ActionResult {
	return ActionResult{
		Success:     true,
		Message:     message,
		FollowUpMsg: followUp,
	}
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

// ReadOnlyAllowlist defines API operations allowed in read-only mode.
// - View actions: always allowed (navigation only)
// - Exec actions: allowed only if Name is in ReadOnlyExecAllowlist
// - API actions: allowed only if Operation is in this list
//
// Security rationale for each allowed operation:
var ReadOnlyAllowlist = map[string]bool{
	// DetectStackDrift: Triggers analysis only, no stack modifications
	"DetectStackDrift": true,
	// InvokeFunctionDryRun: Validation mode, function is not actually invoked
	"InvokeFunctionDryRun": true,
}

var ReadOnlyExecAllowlist = map[string]bool{
	ActionNameSSOLogin: true,
	ActionNameLogin:    true,
}

// IsAllowedInReadOnly returns whether the action can be executed in read-only mode.
func IsAllowedInReadOnly(act Action) bool {
	switch act.Type {
	case ActionTypeExec:
		return ReadOnlyExecAllowlist[act.Name]
	case ActionTypeAPI:
		return ReadOnlyAllowlist[act.Operation]
	default:
		return false
	}
}

// IsExecAllowedInReadOnly checks if an exec action name is allowed in read-only mode.
func IsExecAllowedInReadOnly(actionName string) bool {
	return ReadOnlyExecAllowlist[actionName]
}

// ConfirmTokenName is a helper for ConfirmToken that returns the resource name.
// Use when the action operates on Name rather than ID (e.g., CFN stacks, SFN state machines).
func ConfirmTokenName(r dao.Resource) string {
	return r.GetName()
}

// MinConfirmChars is the minimum number of characters required for dangerous confirmation.
// For tokens longer than this, only the last MinConfirmChars characters need to be typed.
const MinConfirmChars = 6

// ConfirmSuffix returns the suffix of the token that the user must type.
// For empty tokens, returns "CONFIRM" as a fallback to prevent accidental confirmation.
// For tokens <= MinConfirmChars, returns the full token.
// For longer tokens, returns the last MinConfirmChars characters.
func ConfirmSuffix(token string) string {
	if token == "" {
		return "CONFIRM"
	}
	if len(token) <= MinConfirmChars {
		return token
	}
	return token[len(token)-MinConfirmChars:]
}

// ConfirmMatches checks if the user input matches the required confirmation.
// Returns true if input equals the suffix returned by ConfirmSuffix.
func ConfirmMatches(token, input string) bool {
	return input == ConfirmSuffix(token)
}

// Register registers actions for a resource type.
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

// ExecuteWithDAO executes an action with service/resource context for executor lookup.
//
// Exec path conventions:
//   - Interactive (TUI): ActionMenu uses tea.Exec(ExecWithHeader) to suspend TUI
//   - Non-interactive: This function calls executeExec() directly (for programmatic use)
//
// API actions always go through this function → registered executor.
func ExecuteWithDAO(ctx context.Context, action Action, resource dao.Resource, service, resourceType string) ActionResult {
	log.Info("executing action", "action", action.Name, "type", action.Type, "service", service, "resourceType", resourceType, "resourceID", resource.GetID())

	// Validate API action configuration before read-only check (better diagnostics)
	if action.Type == ActionTypeAPI && action.Operation == "" {
		log.Error("API action missing Operation", "action", action.Name, "service", service, "resourceType", resourceType)
		return ActionResult{Success: false, Error: ErrEmptyOperation}
	}

	// Defense-in-depth: UI (NewActionMenu) already filters actions, but re-check here
	// to prevent direct API calls or future code paths from bypassing read-only protection.
	if config.Global().ReadOnly() && !IsAllowedInReadOnly(action) {
		log.Info("read-only denied action", "action", action.Name, "type", action.Type)
		return ActionResult{Success: false, Error: ErrReadOnlyDenied}
	}

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
	if !action.SkipAWSEnv {
		setAWSEnv(execCmd, aws.GetRegionFromContext(ctx))
	}

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
