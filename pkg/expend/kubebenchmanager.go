package expend

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/kubesphere/kubeeye/plugins/plugin-manage/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

//var ClientSet kubernetes.Interface
var ClientSet *kubernetes.Clientset

func InstallPlugin(ctx context.Context, kubeconfig, pluginName string) error {
	var installer Expends
	clients, err := GetK8SClients(kubeconfig)
	if err != nil {
		return err
	}
	installer = Installer{
		CTX:     ctx,
		Clients: clients,
	}
	ClientSet = clients.ClientSet.(*kubernetes.Clientset)

	gvrInfo := pkg.GVRInfo{
		GVR: schema.GroupVersionResource{
			Group:    pkg.PluginGroup,
			Version:  pkg.PluginAPIVersion,
			Resource: pkg.PluginResource,
		},
		Namespace: pkg.PluginNamespaces,
		Name:      pkg.PluginName,
	}
	resources, err := pkg.GetManifest(clients.DynamicClient, gvrInfo)
	if err != nil {
		fmt.Printf("GetManifest err:%s\n", err)
		return err
	}

	KubeBenchCRDResources := []byte(resources)
	KubeBenchCRDResource := bytes.Split(KubeBenchCRDResources, []byte("---"))

	for _, resource := range KubeBenchCRDResource {
		if err := installer.install(resource); err != nil {
			return err
		}
	}

	return nil
}

func UninstallPlugin(ctx context.Context, kubeconfig, pluginName string) error {
	var installer Expends
	clients, err := GetK8SClients(kubeconfig)
	if err != nil {
		return err
	}
	installer = Installer{
		CTX:     ctx,
		Clients: clients,
	}

	gvrInfo := pkg.GVRInfo{
		GVR: schema.GroupVersionResource{
			Group:    pkg.PluginGroup,
			Version:  pkg.PluginAPIVersion,
			Resource: pkg.PluginResource,
		},
		Namespace: pkg.PluginNamespaces,
		Name:      pluginName,
	}
	resources, err := pkg.GetManifest(clients.DynamicClient, gvrInfo)
	if err != nil {
		fmt.Printf("GetManifest err:%s\n", err)
		return err
	}

	KubeBenchCRDResources := []byte(resources)
	KubeBenchCRDResource := bytes.Split(KubeBenchCRDResources, []byte("---"))
	for _, resource := range KubeBenchCRDResource {
		if err := installer.uninstall(resource); err != nil {
			return err
		}
	}

	return nil
}

func IsPluginPodRunning() bool {
	var podName string
	var isRunning bool
	// wait pod creating
	time.Sleep(time.Second * 20)
	pods, err := ClientSet.CoreV1().Pods(pkg.PluginNamespaces).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

FindPodName:
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, "kube-bench-controller") {
			podName = pod.Name
			switch pod.Status.Phase {
			case "Running":
				isRunning = true
			case "ContainerCreating", "Pending":
			default:
				isRunning = false
			}
			break FindPodName
		}
	}
	if !isRunning {
		return tickerGetPodStatus(podName)
	}
	return true
}

//tickerGetPodStatus Query the latest status of pods regularly
func tickerGetPodStatus(podName string) bool {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	isRunning := make(chan bool)
	var count int
	go func(isRunning chan bool) {
		for _ = range ticker.C {
			pod, err := ClientSet.CoreV1().Pods(pkg.PluginNamespaces).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					fmt.Printf("Pod %s in namespace %s not found\n", podName, pkg.PluginNamespaces)
				}
				return
			}

			switch pod.Status.Phase {
			case "Running":
				isRunning <- true
			case "ContainerCreating", "Pending":
			default:
				isRunning <- false
			}
			count++
			if count == pkg.MaxCheckPodCount {
				fmt.Println("ticker timeout...")
				isRunning <- false
			}
		}
	}(isRunning)

	result := <-isRunning
	return result
}
