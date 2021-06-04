package falcon_container_deployer

// Returns the namespace in which the operator runs and creates helper artifacts
func (d *FalconContainerDeployer) Namespace() string {
	return d.Instance.ObjectMeta.Namespace
}
