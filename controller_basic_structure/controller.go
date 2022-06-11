package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"

	appsinformers "k8s.io/client-go/informers/apps/v1"
	listers "k8s.io/client-go/listers/apps/v1"
)

type controller struct {
	clientset      kubernetes.Interface     //interact with K8s
	depLister      listers.DeploymentLister //its a componenet in Informer using which we get the resources
	depCacheSynced cache.InformerSynced     //way to figureout cache has been synced or updated.
	queue          workqueue.RateLimitingInterface
}

func newController(clientset kubernetes.Interface, depInformer appsinformers.DeploymentInformer) *controller {
	c := &controller{
		clientset:      clientset,
		depLister:      depInformer.Lister(),             // informer has Lister
		depCacheSynced: depInformer.Informer().HasSynced, // HasSynced returns true if the shared informer's store has been
		// informed by at least one full LIST of the authoritative state
		// of the informer's object collection.  This is unrelated to "resync".
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
	}

	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handleAdd,
			DeleteFunc: handleDel,
		},
	)

	return c
}

func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("starting controller")
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {
		fmt.Print("waiting for cache to be synced\n")
	}

	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch
}
func (c *controller) worker() {
	fmt.Println(c.depLister.Deployments("minio-api"))

}

func handleAdd(obj interface{}) {
	fmt.Println("add was called")

}

func handleDel(obj interface{}) {
	fmt.Println("del was called")
}
