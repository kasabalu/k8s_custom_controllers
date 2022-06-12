package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

func main() {
	kubeconfig := flag.String("Kubeconfig", "/Users/bkasa724/.kube/config", "location to kube config file")
	//namespace := flag.String("namespace", "default", "namespace name")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	//ns := *namespace
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	ch := make(chan struct{})
	informers := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	c := newController(clientset, informers.Apps().V1().Deployments())

	informers.Start(ch)

	c.run(ch)
	fmt.Println(informers)

}
