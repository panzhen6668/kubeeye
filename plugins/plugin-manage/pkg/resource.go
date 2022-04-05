package pkg
import (
	"context"
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	kubeeyev1alpha1 "github.com/kubesphere/kubeeye/plugins/plugin-manage/api/v1alpha1"
)

var gvr = schema.GroupVersionResource{
	Group:    "kubeeye.kubesphere.io",
	Version:  "v1alpha1",
	Resource: "plugins",
}

type PluginSpec struct {
	Manifest string `json:"manifest,omitempty"`
}

// Plugin is the Schema for the plugins API
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   PluginSpec   `json:"spec,omitempty"`
}

type GVRInfo struct{
	GVR schema.GroupVersionResource `json:"gvr"`
	Namespace string `json:"namespace"`
	Name string `json:"name"`
}
func GetManifest(client dynamic.Interface, gvrInfo GVRInfo) (string, error) {
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
