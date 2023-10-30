package composite

type MissingMetadata struct{}

func (e *MissingMetadata) Error() string {
	return "object does not contain metadata"
}

type InvalidMetadata struct{}

func (e *InvalidMetadata) Error() string {
	return "invalid or empty metadata object"
}

type MissingSpec struct{}

func (e *MissingSpec) Error() string {
	return "object does not contain spec field"
}

type InvalidSpec struct{}

func (e *InvalidSpec) Error() string {
	return "invalid or empty object spec"
}

type WaitingForSpec struct{}

func (w *WaitingForSpec) Error() string {
	return "spec is empty or undefined"
}
