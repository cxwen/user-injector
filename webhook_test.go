package main

import (
	"testing"
	"github.com/golang/glog"
	"encoding/json"
	"fmt"
)

func Test_mutate(t *testing.T) {
	rawStr := `{"kind":"Service","apiVersion":"v1","metadata":{"name":"apache-test","namespace":"default","creationTimestamp":null,"labels":{"name":"apache-svc"},"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"labels\":{\"name\":\"apache-svc\"},\"name\":\"apache-test\",\"namespace\":\"default\"},\"spec\":{\"ports\":[{\"port\":80,\"targetPort\":80}],\"selector\":{\"name\":\"apache\"},\"type\":\"NodePort\"}}\n"}},"spec":{"ports":[{"protocol":"TCP","port":80,"targetPort":80}],"selector":{"name":"apache"},"type":"NodePort","sessionAffinity":"None","externalTrafficPolicy":"Cluster"},"status":{"loadBalancer":{}}}`
	reqRawMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(rawStr), &reqRawMap); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
	}

	var annotationMap interface{}
	var labelMap interface{}

	for k, v := range reqRawMap["metadata"].(map[string]interface{}) {
		fmt.Printf("key: %s, value type: %s\n", k, typeof(v))
		if k != "annotations" && k != "labels" {
			continue
		}
		if k == "annotations" {
			annotationMap = v
		}
		if k == "labels" {
			labelMap = v
		}
	}

	fmt.Println("annotation")
	for k, v := range annotationMap.(map[string]interface{}) {
		fmt.Printf("key: %s   value: %v\n", k, v)
	}

	fmt.Println("label")
	for k, v := range labelMap.(map[string]interface{}) {
		fmt.Printf("key: %s   value: %v\n", k, v)
	}
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

