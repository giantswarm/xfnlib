package composite

// MissingMetadata is raised when an object does not contain a metadata type
type MissingMetadata struct{}

func (e *MissingMetadata) Error() string {
	return "object does not contain metadata"
}

// InvalidMetadata is raised when an object metadata cannot be unpacked to a Metadata object
type InvalidMetadata struct{}

func (e *InvalidMetadata) Error() string {
	return "invalid or empty metadata object"
}

// MissingSpec when an object requiring spec is detected but spec is not found during unpack
type MissingSpec struct{}

func (e *MissingSpec) Error() string {
	return "object does not contain spec field"
}

// InvalidSpec is raised when an object spec cannot be unpacked to the required spec object
type InvalidSpec struct{}

func (e *InvalidSpec) Error() string {
	return "invalid or empty object spec"
}

// WaitingForSpec Raise this error if your input has no spec on the XR but spec is required.
// Methods receiving this should return response.Normal
type WaitingForSpec struct{}

func (w *WaitingForSpec) Error() string {
	return "spec is empty or undefined"
}
