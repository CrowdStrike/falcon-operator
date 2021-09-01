package v1alpha1

func (fc *FalconContainer) SCCEnabled() bool {
	return fc.Spec.GlobalSCC != nil && fc.Spec.GlobalSCC.Enable
}

func (fc *FalconContainer) SCCEnabledRoot() bool {
	return fc.Spec.GlobalSCC != nil && fc.Spec.GlobalSCC.EnableRunAsRoot

}
