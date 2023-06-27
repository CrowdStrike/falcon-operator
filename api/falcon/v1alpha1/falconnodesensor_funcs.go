package v1alpha1

// TargetNs returns a namespace to which the node sensor should be installed to
func (n *FalconNodeSensor) TargetNs() string {
	return "falcon-system"
}
