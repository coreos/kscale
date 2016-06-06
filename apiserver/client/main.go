package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	controllerframework "k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	scaleNS = "scale-ns"
)

func ExitError(msg string, args ...interface{}) {
	fmt.Println("exiting with error:")
	fmt.Printf(msg+"\n", args)
	os.Exit(1)
}

func main() {
	var apisrvAddr string
	var rcNum int
	var podNum int
	var chaosTestOnly bool
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.IntVar(&rcNum, "rc", 1000, "number of RC")
	flag.IntVar(&podNum, "pod", 100, "number of pods per RC")
	flag.BoolVar(&chaosTestOnly, "chaos", false, "running chaos testing only")
	flag.Parse()

	c, err := createClient(apisrvAddr)
	if err != nil {
		ExitError("createClient failed: %v", err)
	}

	fmt.Printf("Creating %d rc, each of %d pods\n", rcNum, podNum)

	if !chaosTestOnly {
		createPods(c, rcNum, podNum)
	}

	// introduce chaos
	var wg sync.WaitGroup
	wg.Add(rcNum)
	for i := 0; i < rcNum; i++ {
		// TODO: clean up on failure
		go func(id int) {
			defer wg.Done()
			deletePodsRandomly(c, id, podNum)
			waitRCRecoverPods(c, id, podNum)
		}(i)
	}
	wg.Wait()
	fmt.Println("Success...")
}

func createPods(c *client.Client, rcNum, podNum int) {
	var wg sync.WaitGroup
	wg.Add(rcNum)
	for i := 0; i < rcNum; i++ {
		// TODO: clean up on failure
		go func(id int) {
			defer wg.Done()
			createRC(c, id, podNum)
			waitRCCreatePods(c, id, podNum)
		}(i)
	}
	wg.Wait()
}

func createRC(c *client.Client, id, podNum int) {
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: makeRCName(id),
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: int32(podNum),
			Selector: makeLabel(id),
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: makeLabel(id),
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "none",
							Image: "none",
						},
					},
				},
			},
		},
	}
	if _, err := c.ReplicationControllers(scaleNS).Create(rc); err != nil {
		ExitError("create RC failed: %v", err)
	}
	fmt.Printf("created rc: %s\n", makeRCName(id))
}

func waitRCCreatePods(c *client.Client, id, podNum int) {
	informer := createPodInformer(c, labels.SelectorFromSet(labels.Set(makeLabel(id))))

	doneCh := make(chan struct{})
	total := 0
	informer.AddEventHandler(controllerframework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			total += 1
			if total == podNum {
				close(doneCh)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	go informer.Run(stopCh)

	store := informer.GetStore()
	start := time.Now()
	for {
		select {
		case <-doneCh:
			if len(store.List()) != podNum {
				panic("unexpected")
			}
			fmt.Printf("rc (%s) created %d pods\n", makeRCName(id), podNum)
			return
		case <-time.After(1 * time.Minute):
			fmt.Printf("%v passed, rc (%s) has created %d pods\n", time.Since(start), makeRCName(id), len(store.List()))
		}
	}
}

func deletePodsRandomly(c *client.Client, id, podNum int) {
	podList, err := c.Pods(scaleNS).List(api.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(makeLabel(id))),
	})
	if err != nil {
		ExitError("list pods failed: %v", err)
	}
	for i, pod := range podList.Items {
		if i%2 != 0 {
			continue
		}
		if err := c.Pods(scaleNS).Delete(pod.Name, api.NewDeleteOptions(0)); err != nil {
			ExitError("delete pod (%s) failed: %v", pod.Name, err)
		}
		fmt.Printf("rc (%s) deleted pod %s\n", makeRCName(id), pod.Name)
	}
}

func waitRCRecoverPods(c *client.Client, id, podNum int) {
	informer := createPodInformer(c, labels.SelectorFromSet(labels.Set(makeLabel(id))))
	store := informer.GetStore()

	doneCh := make(chan bool, 1)
	informer.AddEventHandler(controllerframework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if len(store.List()) == podNum {
				select {
				case doneCh <- true:
				default:
				}
			}
		},
	})
	stopCh := make(chan struct{})
	defer close(stopCh)
	go informer.Run(stopCh)

	if len(store.List()) == podNum {
		fmt.Printf("rc (%s) recovered to %d pods\n", makeRCName(id), podNum)
		return
	}
	start := time.Now()
	for {
		select {
		case <-doneCh:
			fmt.Printf("rc (%s) recovered to %d pods\n", makeRCName(id), podNum)
			return
		case <-time.After(1 * time.Minute):
			fmt.Printf("%v passed, rc (%s) has created %d pods\n", time.Since(start), makeRCName(id), len(store.List()))
		}
	}

}

func createPodInformer(c *client.Client, labelSelector labels.Selector) controllerframework.SharedInformer {
	informer := controllerframework.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labelSelector
				return c.Pods(scaleNS).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return c.Pods(scaleNS).Watch(options)
			},
		},
		&api.Pod{},
		0,
	)
	return informer
}

func createClient(addr string) (*client.Client, error) {
	cfg := &restclient.Config{
		Host:  fmt.Sprintf("http://%s", addr),
		QPS:   1000,
		Burst: 1000,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func makeRCName(id int) string {
	return fmt.Sprintf("scale-rc-%d", id)
}

func makeLabel(id int) map[string]string {
	return map[string]string{"name": fmt.Sprintf("scale-label-%d", id)}
}
