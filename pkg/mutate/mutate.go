// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mutate mutates
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod
	var job *batchv1.Job
	var patch map[string]interface{}
	var testVol bool
	testVol = false

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if ar.Kind.Kind == "Pod" {
			if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
				return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
			}
		} else if ar.Kind.Kind == "Job" {
			if err := json.Unmarshal(ar.Object.Raw, &job); err != nil {
				return nil, fmt.Errorf("unable unmarshal job json object %v", err)
			}
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"mutateme": "yup it did it",
		}

		// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
		// tell K8S how it should modifiy it
		// p := []map[string]string{}
		// for range pod.Spec.Containers {
		// 	// patch := map[string]string{
		// 	// 	"op":    "replace",
		// 	// 	"path":  fmt.Sprintf("/spec/containers/%d/image", i),
		// 	// 	"value": "debian",
		// 	// }
		// 	// add a volume to the pod

		// 	p = append(p, patch)
		// }
		// add patch to add a volume to the pod
		p := []map[string]interface{}{}
		if ar.Kind.Kind == "Pod" {
			for i := range pod.Spec.Volumes {
				if pod.Spec.Volumes[i].Name == "ca-certificates" {
					testVol = true
				}
			}
			if !testVol {
				patch = map[string]interface{}{
					"op":   "add",
					"path": "/spec/volumes/-",
					// "value": map[string]interface{}{"name": "test-volume", "emptyDir": map[string]interface{}{"sizeLimit": "0"}},
					"value": map[string]interface{}{"name": "ca-certificates", "configMap": map[string]interface{}{"name": "trust-bundle", "items": []map[string]interface{}{{"key": "trust-bundle.pem", "path": "ca-certificates.crt"}}, "defaultMode": 420}},
				}
				p = append(p, patch)

				for i := range pod.Spec.InitContainers {
					// ensure volumeMounts array exists
					if pod.Spec.InitContainers[i].VolumeMounts == nil {
						patch = map[string]interface{}{
							"op":    "add",
							"path":  fmt.Sprintf("/spec/initContainers/%d/volumeMounts", i),
							"value": []corev1.VolumeMount{},
						}
						p = append(p, patch)
					}
					// add a volume mount to the init container
					patch = map[string]interface{}{
						"op":   "add",
						"path": fmt.Sprintf("/spec/initContainers/%d/volumeMounts/-", i),
						"value": map[string]interface{}{
							"name":      "ca-certificates",
							"mountPath": "/etc/ssl/certs/",
							"readOnly":  true,
						},
					}
					p = append(p, patch)
				}
			}

			testVol = false
			for i := range pod.Spec.Containers {
				// add a volume mount to the container
				// add volume mount only if it doesn't already exist
				for j := range pod.Spec.Containers[i].VolumeMounts {
					if pod.Spec.Containers[i].VolumeMounts[j].Name == "ca-certificates" {
						testVol = true
					}
				}
				if !testVol {
					patch = map[string]interface{}{
						"op":   "add",
						"path": fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
						// "value": map[string]interface{}{"name": "test-volume", "mountPath": "/test-volume"},
						"value": map[string]interface{}{"name": "ca-certificates", "mountPath": "/etc/ssl/certs/", "readOnly": true},
					}
					p = append(p, patch)
					// add additional volume mounts to suse containers -> /var/lib/ca-certificates/ca-bundle.pem
					patch = map[string]interface{}{
						"op":   "add",
						"path": fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
						// "value": map[string]interface{}{"name": "test-volume", "mountPath": "/test-volume"},
						"value": map[string]interface{}{"name": "ca-certificates", "mountPath": "/var/lib/ca-certificates/ca-bundle.pem", "subPath": "ca-certificates.crt", "readOnly": true},
					}
					p = append(p, patch)
				}
			}
		} else if ar.Kind.Kind == "Job" {
			// ensure volumes array exists
			if job.Spec.Template.Spec.Volumes == nil {
				patch = map[string]interface{}{
					"op":    "add",
					"path":  "/spec/template/spec/volumes",
					"value": []corev1.Volume{},
				}
				p = append(p, patch)
			}

			patch = map[string]interface{}{
				"op":    "add",
				"path":  "/spec/template/spec/volumes/-",
				"value": map[string]interface{}{"name": "ca-certificates", "configMap": map[string]interface{}{"name": "trust-bundle", "items": []map[string]interface{}{{"key": "trust-bundle.pem", "path": "ca-certificates.crt"}}, "defaultMode": 420}},
			}
			p = append(p, patch)
			for i := range job.Spec.Template.Spec.Containers {
				// ensure volumeMounts array exists
				if job.Spec.Template.Spec.Containers[i].VolumeMounts == nil {
					patch = map[string]interface{}{
						"op":    "add",
						"path":  fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts", i),
						"value": []corev1.VolumeMount{},
					}
					p = append(p, patch)
				}
				// add a volume mount to the container
				patch = map[string]interface{}{
					"op":    "add",
					"path":  fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts/-", i),
					"value": map[string]interface{}{"name": "ca-certificates", "mountPath": "/etc/ssl/certs/", "readOnly": true},
				}
				p = append(p, patch)
			}

			// add additional volume mounts to suse containers -> /var/lib/ca-certificates/ca-bundle.pem
			for i := range job.Spec.Template.Spec.Containers {
				// ensure volumeMounts array exists
				if job.Spec.Template.Spec.Containers[i].VolumeMounts == nil {
					patch = map[string]interface{}{
						"op":    "add",
						"path":  fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts", i),
						"value": []corev1.VolumeMount{},
					}
					p = append(p, patch)
				}
				// add a volume mount to the container
				patch = map[string]interface{}{
					"op":    "add",
					"path":  fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts/-", i),
					"value": map[string]interface{}{"name": "ca-certificates", "mountPath": "/var/lib/ca-certificates/ca-bundle.pem", "subPath": "ca-certificates.crt", "readOnly": true},
				}
				p = append(p, patch)
			}
		}
		// parse the []map into JSON
		resp.Patch, _ = json.Marshal(p)

		// Success, of course ;)
		resp.Result = &metav1.Status{
			Status: "Success",
		}

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
