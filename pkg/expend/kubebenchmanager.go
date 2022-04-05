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
	"net/http"
	"strings"
	"time"
)

var ClientSet kubernetes.Interface

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
	ClientSet = clients.ClientSet
	fmt.Printf("InstallKubeBenchInstallKubeBenchInstallKubeBench\n")
	KubeBenchCRDUrl := "http://139.198.27.46:30088/kubeeye-plugins-kubebench.yaml" //"https://raw.githubusercontent.com/panzhen6668/go.uuid/main/kubeeye-kube-bench.yaml"
	//KubeBenchResourceUrl := "./plugins/kube-bench/config/samples/kubeeye_v1alpha1_kubebench.yaml"

	KubeBenchCRD, err := http.Get(KubeBenchCRDUrl)
	if err != nil {
		return err
	}
	defer KubeBenchCRD.Body.Close()

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
	//fmt.Printf("GetManifest :%s\n", resources)
	KubeBenchCRDResources := []byte(resources)
	//KubeBenchCRDResources, err = ioutil.ReadAll(KubeBenchCRD.Body)
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

//IsPluginPodRunning
func IsPluginPodRunning() bool {
	var podName string
	ch := make(chan int)
	go func() {
		pods, err := ClientSet.CoreV1().Pods(pkg.PluginNamespaces).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, "kube-bench-controller") {
				if pod.Status.Phase == "Running" {
					ch <- 1
				}
				podName = pod.Name
				fmt.Println("STATUS.Phase:" + pod.Status.Phase)
				break
			}
		}
		for {
			time.Sleep(time.Second * 8)
			if po, err := ClientSet.CoreV1().Pods(pkg.PluginNamespaces).Get(context.TODO(), podName, metav1.GetOptions{}); err != nil {
				if errors.IsNotFound(err) {
					fmt.Printf("Pod %s in namespace %s not found\n", podName, pkg.PluginNamespaces)
				}
				switch po.Status.Phase {
				case "Running":
					ch <- 1
				case "ContainerCreating":
					fmt.Println("ContainerCreating")
				default:
					ch <- 3
				}
			}
		}
		fmt.Printf("4444444444444444 ..................")
	}()
	select {
	case res := <-ch:
		fmt.Println(res)
		fmt.Printf("Plugin Pod Running ..................")
		return true
	case <-time.After(time.Second * 120): //设置超时时间
		fmt.Println("timeout !!!!!!!!!!!!!!")
		return false
	}
}
