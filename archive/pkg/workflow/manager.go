package workflow

type WorkflowManager struct {
	chanAcceptWorkflow chan *Workflow
	workflows          map[string]*Workflow
}

func (wm *WorkflowManager) GetChanAcceptWorkflow() chan *Workflow {
	return wm.chanAcceptWorkflow
}

func NewWorkflowManager() *WorkflowManager {
	return &WorkflowManager{
		chanAcceptWorkflow: make(chan *Workflow),
		workflows:          make(map[string]*Workflow),
	}
}
