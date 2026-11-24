package forge

import "context"

// WorkflowDispatcher starts one run of a workflow that has a workflow_dispatch trigger.
type WorkflowDispatcher interface {
	WorkflowDispatch(ctx context.Context, file, ref string) error
}
