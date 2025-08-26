package workflow

import (
	v1alpha1 "github.com/numaproj/numaflow/pkg/apis/numaflow/v1alpha1"
)

type Workflow struct {
	// provisionSpec
	deploymentSpec v1alpha1.Pipeline
	// controlSpec
}
