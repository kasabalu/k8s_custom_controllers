package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"

	"time"
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
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cntroller-test"),
	}

	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		},
	)

	return c
}

func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("starting controller")
	// Informer maintain cache, making sure Informer cache synced successfully
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {
		fmt.Println("waiting for cache to be synced")
	}

	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch
}
func (c *controller) worker() {
	//fmt.Println(c.depLister.Deployments("minio-api"))

	for c.processItem() {
		// This loop will terminate when processItem return False.
		//worker func will called every one sec from Util until channel is closed.
	}

}

func (c *controller) processItem() bool {
	item, shutdown := c.queue.Get() // getting the obj from Queue
	if shutdown {
		return false
	}

	defer c.queue.Forget(item)                   // this will delete/mark as proecces item from queue, make sure not processed again.
	key, err := cache.MetaNamespaceKeyFunc(item) //
	if err != nil {
		fmt.Printf("getting key from cahce %s\n", err.Error())
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key) // getting namespace and name
	if err != nil {
		fmt.Printf("splitting key into namespace and name %s\n", err.Error())
		return false
	}
	err = c.syncDeployment(ns, name)
	if err != nil {
		// re-try
		fmt.Printf("syncing deployment %s\n", err.Error())
		return false
	}
	return true

}

func (c *controller) syncDeployment(ns, name string) error {
	ctx := context.Background()

	dep, err := c.depLister.Deployments(ns).Get(name)
	if err != nil {
		fmt.Printf("getting deployment from lister %s\n", err.Error())
	}
	// create service
	// we have to modify this, to figure out the port
	// our deployment's container is listening on
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: depLabels(*dep), // getting lables for a particular deployment using dep obj
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}

	_, err = c.clientset.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("creating service %s\n", err.Error())
	}
	// create ingress
	return nil
}

func depLabels(dep appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}

func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("add was called")
	c.queue.Add(obj) //adding to queue to process

}

func (c *controller) handleDel(obj interface{}) {
	fmt.Println("del was called")
	c.queue.Add(obj) //adding to queue to process

}
