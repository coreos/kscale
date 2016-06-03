package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const numPods = 1000000

func ExitError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args)
	os.Exit(1)
}

func main() {
	var apisrvAddr string
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.Parse()

	cfg := &restclient.Config{
		Host:  fmt.Sprintf("http://%s", apisrvAddr),
		QPS:   1000,
		Burst: 1000,
	}
	c, err := client.New(cfg)
	if err != nil {
		ExitError("client.New failed: %v", err)
	}

	// createRC(c)
	// updateRC(c)
	// deleteRC(c)
	start := 0
	for {
		time.Sleep(3 * time.Second)
		podList, err := c.Pods(api.NamespaceAll).List(api.ListOptions{})
		if err != nil {
			ExitError("List pods failed: %v", err)
		}
		start += 3
		fmt.Println(start, len(podList.Items))
		if len(podList.Items) == numPods {
			break
		}
	}
	fmt.Println("Success...")
	// c.Pods("wan-ns").Delete("wan-rc-tort0", api.NewDeleteOptions(0))
}

func createRC(c *client.Client) {
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: "wan-rc",
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: numPods,
			Selector: map[string]string{
				"name": "wan-label",
			},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"name": "wan-label"},
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
	if _, err := c.ReplicationControllers("wan-ns").Create(rc); err != nil {
		ExitError("create RC failed: %v", err)
	}
	fmt.Println("created rc...")
}

func updateRC(c *client.Client) {
	rc := &api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: "wan-rc",
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 0,
			Selector: map[string]string{
				"name": "wan-label",
			},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"name": "wan-label"},
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
	if _, err := c.ReplicationControllers("wan-ns").Update(rc); err != nil {
		ExitError("update RC failed: %v", err)
	}
	fmt.Println("updated rc...")
}

func deleteRC(c *client.Client) {
	if err := c.ReplicationControllers("wan-ns").Delete("wan-rc"); err != nil {
		ExitError("create RC failed: %v", err)
	}
}