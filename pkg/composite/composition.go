package composite

import (
	"reflect"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Composition contains the main request objects required for interacting with composition function resources.
type Composition struct {
	// ObservedComposite is an object that reflects the composite resource that is created from the claim
	ObservedComposite any

	// DesiredComposite is the raw composite resource we want creating
	DesiredComposite *resource.Composite

	// ObservedComposed is a set of resources that are composed by the composite and exist in the cluster
	ObservedComposed map[resource.Name]resource.ObservedComposed

	// DesiredComposed is the set of resources we require to be created
	DesiredComposed map[resource.Name]*resource.DesiredComposed

	// Input is the information brought in from the function binding
	Input InputProvider
}

// InputProvider This is basically a wrapper to `runtime.Object` and exists to ensure that
// all inputs to the `New` conform to a supported type
type InputProvider interface {
	runtime.Object
}

// New takes a RunFunctionRequest object and converts it to a Composition
//
// This method should be called at the top of your RunFunction.
//
// Example:
//
//	func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (rsp *fnv1beta1.RunFunctionResponse, err error) {
//		f.log.Info("Running Function", composedName, req.GetMeta().GetTag())
//		rsp = response.To(req, response.DefaultTTL)
//
//		input := v1beta1.Input{}
//		if f.composed, err = composite.New(req, &input, &f.composite); err != nil {
//			response.Fatal(rsp, errors.Wrap(err, "error setting up function "+composedName))
//			return rsp, nil
//		}
//
//		...
//		// Function body
//		...
//
//		if err = f.composed.ToResponse(rsp); err != nil {
//			response.Fatal(rsp, errors.Wrapf(err, "cannot convert composition to response %T", rsp))
//			return
//		}
//
//		response.Normal(rsp, "Successful run")
//		return rsp, nil
//	}
func New(req *fnv1beta1.RunFunctionRequest, input InputProvider, composite any) (c *Composition, err error) {
	c = &Composition{
		Input:             input,
		ObservedComposite: composite,
	}

	if c.DesiredComposite, err = request.GetDesiredCompositeResource(req); err != nil {
		err = errors.Wrapf(err, "cannot get desired composed resources from %T", req)
		return
	}

	var oxr *resource.Composite
	if oxr, err = request.GetObservedCompositeResource(req); err != nil {
		err = errors.Wrap(err, "cannot get observed composite resource")
		return
	}

	if err = To(oxr.Resource.Object, &c.ObservedComposite); err != nil {
		err = errors.Wrapf(err, "Failed to convert XR object to struct %T", c.ObservedComposite)
		return
	}

	c.DesiredComposite.Resource.SetAPIVersion(oxr.Resource.GetAPIVersion())
	c.DesiredComposite.Resource.SetKind(oxr.Resource.GetKind())

	if c.DesiredComposed, err = request.GetDesiredComposedResources(req); err != nil {
		err = errors.Wrapf(err, "cannot get desired composite resources from %T", req)
		return
	}

	if c.ObservedComposed, err = request.GetObservedComposedResources(req); err != nil {
		err = errors.Wrapf(err, "cannot get observed composed resources from %T", req)
		return
	}

	if err = request.GetInput(req, c.Input); err != nil {
		return
	}

	return
}

// ToResponse converts the composition back into the response object
//
// This method should be called at the end of your RunFunction immediately before returning a normal response.
// Wrap this in an error handler and set `response.Fatal` on error
func (c *Composition) ToResponse(rsp *fnv1beta1.RunFunctionResponse) (err error) {
	if err = response.SetDesiredCompositeResource(rsp, c.DesiredComposite); err != nil {
		err = errors.Wrapf(err, "cannot set desired composite resources in %T", rsp)
		return
	}

	if err = response.SetDesiredComposedResources(rsp, c.DesiredComposed); err != nil {
		err = errors.Wrapf(err, "cannot set desired composed resources in %T", rsp)
	}
	return
}

// AddDesired takes an unstructured object and adds it to the desired composed resources
//
// If the object exists on the stack already, we do a deepEqual to see if the object has changed
// and if not, this method won't do anything.
func (c *Composition) AddDesired(name string, object *unstructured.Unstructured) (err error) {
	if o, ok := c.DesiredComposed[resource.Name(name)]; ok {
		// Object exists and hasn't changed
		if reflect.DeepEqual(o.Resource.Object, object.Object) {
			return
		}
	}
	c.DesiredComposed[resource.Name(name)] = &resource.DesiredComposed{
		Resource: &composed.Unstructured{
			Unstructured: *object,
		},
		Ready: resource.ReadyTrue,
	}
	return
}
