package pkg

import (
	"context"
	"encoding/json"
	kubeeyev1alpha1 "github.com/kubesphere/kubeeye/plugins/plugin-manage/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type GVRInfo struct {
	GVR       schema.GroupVersionResource `json:"gvr"`
	Namespace string                      `json:"namespace"`
	Name      string                      `json:"name"`
}

func GetPluginManifest(client dynamic.Interface, gvrInfo GVRInfo) (string, error) {
	utd, err := client.Resource(gvrInfo.GVR).Namespace(gvrInfo.Namespace).Get(context.TODO(), gvrInfo.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	data, err := utd.MarshalJSON()
	if err != nil {
		return "", err
	}
	//var ct Plugin
	var ct kubeeyev1alpha1.Plugin
	if err := json.Unmarshal(data, &ct); err != nil {
		return "", err
	}
	return ct.Spec.Manifest, nil
}
