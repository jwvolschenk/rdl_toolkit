package rdl

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
)

// ErrorCode is a stable identifier agents can branch on.
type ErrorCode string

const (
	CodeArgMissing       ErrorCode = "ARG_MISSING"
	CodeArgInvalid       ErrorCode = "ARG_INVALID"
	CodeFileNotFound     ErrorCode = "FILE_NOT_FOUND"
	CodeXMLParse         ErrorCode = "XML_PARSE"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeIndexOutOfRange  ErrorCode = "INDEX_OUT_OF_RANGE"
	CodePrecondition     ErrorCode = "PRECONDITION"
	CodeIO               ErrorCode = "IO_ERROR"
)

// AgentError is a structured error for MCP and agent consumers.
type AgentError struct {
	Code    ErrorCode
	Message string
	Hint    string
	Context map[string]any
	cause   error
}

func (e *AgentError) Error() string { return e.Message }
func (e *AgentError) Unwrap() error { return e.cause }

// AsAgentError returns the AgentError if err wraps one.
func AsAgentError(err error) (*AgentError, bool) {
	var ae *AgentError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// MapLoadError converts Load/save failures into AgentErrors when possible.
func MapLoadError(err error, path string) error {
	if err == nil {
		return nil
	}
	if ae, ok := AsAgentError(err); ok {
		return ae
	}
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
		return &AgentError{
			Code:    CodeFileNotFound,
			Message: fmt.Sprintf("file not found: %s", path),
			Context: map[string]any{"path": path},
			cause:   err,
		}
	}
	if strings.Contains(err.Error(), "parsing XML") {
		return &AgentError{
			Code:    CodeXMLParse,
			Message: fmt.Sprintf("malformed XML in %s", path),
			Context: map[string]any{"path": path, "detail": err.Error()},
			cause:   err,
		}
	}
	if strings.Contains(err.Error(), "reading file") {
		return &AgentError{
			Code:    CodeFileNotFound,
			Message: fmt.Sprintf("cannot read file: %s", path),
			Context: map[string]any{"path": path},
			cause:   err,
		}
	}
	return err
}

// NewNotFoundError builds a NOT_FOUND error with available names when known.
func NewNotFoundError(element, name string, available []string) *AgentError {
	ctx := map[string]any{"element": element, "name": name}
	if len(available) > 0 {
		ctx["available"] = available
	}
	return &AgentError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s %q not found", element, name),
		Hint:    "call the matching rdl_list_* tool to see what exists",
		Context: ctx,
	}
}

// NewIndexError builds an INDEX_OUT_OF_RANGE error.
func NewIndexError(kind string, index, min, max int) *AgentError {
	return &AgentError{
		Code:    CodeIndexOutOfRange,
		Message: fmt.Sprintf("%s index %d out of range (valid: %d..%d)", kind, index, min, max),
		Context: map[string]any{"index": index, "min": min, "max": max, "kind": kind},
	}
}

// NewPreconditionError builds a PRECONDITION error.
func NewPreconditionError(message, hint string) *AgentError {
	return &AgentError{
		Code:    CodePrecondition,
		Message: message,
		Hint:    hint,
	}
}

// NewArgInvalidError builds an ARG_INVALID error.
func NewArgInvalidError(param, message, expected string) *AgentError {
	ctx := map[string]any{"param": param}
	if expected != "" {
		ctx["expected"] = expected
	}
	return &AgentError{
		Code:    CodeArgInvalid,
		Message: message,
		Context: ctx,
	}
}

func (d *Document) dataSourceNames() []string {
	return namedElementList(d, "DataSource")
}

func (d *Document) dataSetNames() []string {
	return namedElementList(d, "DataSet")
}

func (d *Document) parameterNames() []string {
	return namedElementList(d, "ReportParameter")
}

func (d *Document) tablixNames() []string {
	return namedElementList(d, "Tablix")
}

func namedElementList(d *Document, tag string) []string {
	nodes := findAllNamed(d.root, tag)
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if name := n.SelectAttr("Name"); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func findAllNamed(root *xmlquery.Node, tag string) []*xmlquery.Node {
	if root == nil {
		return nil
	}
	// xmlquery is already imported in other files; use Find
	return xmlquery.Find(root, "//"+tag)
}
