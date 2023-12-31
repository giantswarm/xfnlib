package composite

import (
	"encoding/json"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ToUnstructured is a helper function that creates an unstructured object from
// any object that contains metadata, spec and optionally status.
func ToUnstructured(apiVersion, kind, object any) (u *unstructured.Unstructured, err error) {
	u = &unstructured.Unstructured{}
	type objS struct {
		Metadata map[string]interface{}
		Spec     map[string]interface{}
		Status   map[string]interface{}
	}
	var o objS
	if err = To(object, &o); err != nil {
		return
	}

	if len(o.Metadata) == 0 {
		err = &InvalidMetadata{}
		return
	}

	if len(o.Spec) == 0 {
		err = &InvalidSpec{}
		return
	}

	u.Object = map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   o.Metadata,
		"spec":       o.Spec,
	}
	if len(o.Status) > 0 {
		u.Object["status"] = o.Status
	}
	return
}

// ToUnstructuredKubernetesObject is a helper function that wraps a given CR
// resource in a `crossplane-contrib/provider-kubernetes.Object` structure and
// returns this as an unstructured.Unstructured object
//
// mr any The managed resource to wrap
// providerConfigRef string
func ToUnstructuredKubernetesObject(mr any, providerConfigRef, deletionPolicy string) (o *unstructured.Unstructured, err error) {
	o = &unstructured.Unstructured{}
	var ud map[string]interface{} // unstructured data
	if err = To(mr, &ud); err != nil {
		return
	}

	if _, ok := ud["metadata"]; !ok {
		err = errors.Wrap(&MissingMetadata{}, "unable to create kubernetes object")
		return
	}

	var meta metav1.ObjectMeta
	if err = To(ud["metadata"], &meta); err != nil {
		err = errors.Wrapf(err, "unable to create kubernetes object %+v", ud["metadata"])
		return
	}

	var labels map[string]interface{} = make(map[string]interface{})
	for k, v := range meta.Labels {
		labels[k] = v
	}

	o.Object = map[string]interface{}{
		"apiVersion": "kubernetes.crossplane.io/v1alpha1",
		"kind":       "Object",
		"metadata": map[string]interface{}{
			"name":   meta.Name,
			"labels": labels,
		},
		"spec": map[string]interface{}{
			"deletionPolicy": deletionPolicy,
			"forProvider": map[string]interface{}{
				"manifest": ud,
			},
			"writeConnectionSecretToRef": map[string]interface{}{
				"name":      meta.Name,
				"namespace": meta.Namespace,
			},
			"providerConfigRef": map[string]interface{}{
				"name": providerConfigRef,
			},
		},
	}
	return
}

// To is a helper function that converts any object to any object by sending it
// round-robin through `json.Marshal`
func To(resource any, jsonObject any) (err error) {
	var b []byte
	if b, err = json.Marshal(resource); err != nil {
		return
	}

	err = json.Unmarshal(b, jsonObject)
	return
}
