package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
	"os"
	"strings"
)

const (
	ingoreUserNamePrefix   = "system:serviceaccount"
	defaultInjectionSuffix = "cxwen.com"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

type WebhookServer struct {
	server *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	sidecarCfgFile string // path to sidecar injector configuration file
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		if ! strings.HasPrefix(ar.Request.UserInfo.Username, ingoreUserNamePrefix) {
			fmt.Println(r.URL.Path)
			if r.URL.Path == "/mutate" {
				admissionResponse = whsvr.mutate(&ar)
			}
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	} else {
		if ! strings.HasPrefix(ar.Request.UserInfo.Username, ingoreUserNamePrefix) {
			glog.Infof("Write reponse success ...")
		}
	}
}

func createpatchOperation(target map[string]string, added map[string]string, updateType string) (patch []patchOperation) {
	if target == nil {
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  "/metadata/" + updateType,
			Value: added,
		})
		return
	}

	for key, value := range added {
		if target[key] == "" {
			target[key] = value
		}
	}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  "/metadata/" + updateType,
		Value: target,
	})

	return
}

func createPatch(availableAnnotations map[string]string, annotations map[string]string, availableLabels map[string]string, labels map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, createpatchOperation(availableAnnotations, annotations, "annotations")...)
	patch = append(patch, createpatchOperation(availableLabels, labels, "labels")...)

	return json.Marshal(patch)
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	var (
		resourceName string
	)

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	reqRawMap := make(map[string]interface{})
	if err := json.Unmarshal(req.Object.Raw, &reqRawMap); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	injectionType := os.Getenv("INJECTIONT_TYPE")
	if injectionType == "" {
		injectionType = "label"
	}

	availableAnnotations := make(map[string]string)
	availableLabels := make(map[string]string)

	typeArr := strings.Split(injectionType, ",")
	for k, v := range reqRawMap["metadata"].(map[string]interface{}) {
		if k != "annotations" && k != "labels" {
			continue
		}
		if k == "annotations" {
			for k, v := range v.(map[string]interface{}) {
				availableAnnotations[k] = fmt.Sprintf("%v", v)
			}
		}

		if k == "labels" {
			for k, v := range v.(map[string]interface{}) {
				availableLabels[k] = fmt.Sprintf("%v", v)
			}
		}

	}

	annotations := make(map[string]string)
	addLabels := make(map[string]string)

	injectionSuffix := os.Getenv("INJECTION_SUFFIX")
	if injectionSuffix == "" {
		injectionSuffix = defaultInjectionSuffix
	}
	injectionKey := fmt.Sprintf("username.%s", injectionSuffix)

	for _, t := range typeArr {
		if t == "annotation" {
			annotations[injectionKey] = req.UserInfo.Username
		}

		if t == "label" {
			addLabels[injectionKey] = req.UserInfo.Username
		}
	}

	patchBytes, err := createPatch(availableAnnotations, annotations, availableLabels, addLabels)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
