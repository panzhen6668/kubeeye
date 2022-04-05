package expend

import (
	"bytes"
	"context"
	"fmt"
	"github.com/kubesphere/kubeeye/pkg/kube"
	"log"

	//"github.com/lithammer/dedent"
	"github.com/pkg/errors"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

func ResourceCreater(clients *kube.KubernetesClient, ctx context.Context, resource []byte) (err error) {
	dynamicClient := clients.DynamicClient
	clientset := clients.ClientSet

	// Parse Resources,get the unstructured resource
	mapping, unstructuredResource, err := ParseResources(clientset, resource)
	if err != nil {
		return err
	}

	// create resource
	if err := CreateResource(ctx, dynamicClient, mapping, unstructuredResource); err != nil {
		return err
	}

	return nil
}

func CreateResource(ctx context.Context, dynamicClient dynamic.Interface, mapping *meta.RESTMapping, unstructuredResource *unstructured.Unstructured,) error {
	// get namespace from resource.Object
	namespace := unstructuredResource.GetNamespace()
	fmt.Printf("!!! namespace:%s \n", namespace)
	if namespace == ""{
		namespace = "kubeeye-system"
	}
	result, err := dynamicClient.Resource(mapping.Resource).Namespace(namespace).Create(ctx, unstructuredResource, metav1.CreateOptions{})
	if err != nil {
		if kubeErr.IsAlreadyExists(err) {
			return nil
		} else if kubeErr.IsInvalid(err) {
			return errors.Wrap(err, "Create resource failed, resource is invalid")
		}
	}
	fmt.Printf("!!!!!!!!!!!! result:%v", result)
	//fmt.Printf("%s\t%s\t created\n", result.GetKind(), result.GetName())
	return nil
}

func ResourceRemover(clients *kube.KubernetesClient, ctx context.Context, resource []byte) (err error) {
	clientset := clients.ClientSet
	dynamicClient := clients.DynamicClient

	mapping, unstructuredResource, err := ParseResources(clientset, resource)
	if err != nil {
		return err
	}

	if err := RemoveResource(ctx, dynamicClient, mapping, unstructuredResource); err != nil {
		return err
	}

	return nil
}

func RemoveResource(ctx context.Context, dynamicClient dynamic.Interface, mapping *meta.RESTMapping,unstructuredResource *unstructured.Unstructured) error {
	name := unstructuredResource.GetName()
	namespace := unstructuredResource.GetNamespace()

	// delete resource by dynamic client
	if err := dynamicClient.Resource(mapping.Resource).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if kubeErr.IsNotFound(err) {
			return nil
		} else {
			return errors.Wrap(err, "failed to remove resource")
		}
	}
	fmt.Printf("%s\t%s\t deleted\n", unstructuredResource.GetKind(), name)
	return nil
}

// ParseResources by parsing the resource, put the result into unstructuredResource and return it.
func ParseResources(clientset kubernetes.Interface, resource []byte) (*meta.RESTMapping, *unstructured.Unstructured, error) {
	var unstructuredResource unstructured.Unstructured
	fmt.Printf("!!!!!!!!!!!!!!!!!!!!!!!!! resource:%+v\n",string(resource))
	//r := dedent.Dedent(string(resource))
	// decode resource for convert the resource to unstructur.
	//newreader := strings.NewReader(bytes.NewReader(r))
	decode := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(resource), 4096)
	// get resource kind and group
	disc := clientset.Discovery()
	restMapperRes, _ := restmapper.GetAPIGroupResources(disc)
	restMapper := restmapper.NewDiscoveryRESTMapper(restMapperRes)
	ext := runtime.RawExtension{}
	if err := decode.Decode(&ext); err != nil {
		return nil, &unstructuredResource, errors.Wrap(err, "decode error")
	}
	// get resource.Object
	//obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, nil)
	obj, gvk, err :=  yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(ext.Raw, nil, nil)
	if err != nil {
		return nil, &unstructuredResource, errors.Wrap(err, "failed to get resource object")
	}
	// identifies a preferred resource mapping
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, &unstructuredResource, errors.Wrap(err, "failed to get resource mapping")
	}

	// convert the resource.Object into unstructured

	unstructuredResource.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	fmt.Printf("!!!!!!!!!!!!!!!!!!!!!!!!! unstructuredResource:%+v\n",unstructuredResource)
	if err != nil {
		return nil, &unstructuredResource, errors.Wrap(err, "failed to converts an object into unstructured representation")
	}
	return mapping, &unstructuredResource, nil
}

func InstallV2(nameSpace string,clientSet kubernetes.Interface,dd dynamic.Interface,filebytes []byte){
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 4096)
	//for {
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		fmt.Printf("Decode err:%v \n",err)
		//break
	}

	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		log.Fatal(err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	fmt.Printf("!!!!!!!!!!!!!!!!!!!!!!!!! unstructuredMap:%+v\n",unstructuredObj)

	gr, err := restmapper.GetAPIGroupResources(clientSet.Discovery())
	if err != nil {
		log.Fatal(err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		log.Fatal(err)
	}

	var dri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if unstructuredObj.GetNamespace() == "" {
			fmt.Printf("SetNamespace  \n")
			unstructuredObj.SetNamespace(nameSpace)
		}
		fmt.Printf("Resource !!!!!!!!!!!!!!!11111111111111111  \n")
		dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	} else {
		fmt.Printf("Resource !!!!!!!!!!!!!!!2222222222222222  \n")
		dri = dd.Resource(mapping.Resource)
	}
	//dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	fmt.Printf("Resource !!!!!!!!!!!!!!!3333333333333333333  \n")
	obj2, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("!!!!!!!!!!!!!!!! dri.Create err:%v\n", err)
		if kubeErr.IsAlreadyExists(err) {
			fmt.Printf("IsAlreadyExists  \n")
			return
		} else if kubeErr.IsInvalid(err) {
			fmt.Printf("IsInvalid  \n")
			return
		}
	}else{
		fmt.Printf("%s/%s created\n", obj2.GetKind(), obj2.GetName())
	}
	return
	//}
}