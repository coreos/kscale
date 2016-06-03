package main

import (
	"flag"
	"fmt"

	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/controller"
	replicationcontroller "k8s.io/kubernetes/pkg/controller/replication"
	"k8s.io/kubernetes/pkg/util/wait"
)

func main() {
	lookupCacheSizeForRC := 4096
	concurrentRCSyncs := 5
	var apisrvAddr string
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.Parse()

	kubeconfig := &restclient.Config{
		Host:  fmt.Sprintf("http://%s", apisrvAddr),
		QPS:   1000,
		Burst: 1000,
	}

	replicationcontroller.NewReplicationManagerFromClient(
		clientset.NewForConfigOrDie(restclient.AddUserAgent(kubeconfig, "replication-controller")),
		controller.NoResyncPeriodFunc,
		replicationcontroller.BurstReplicas,
		lookupCacheSizeForRC,
	).Run(concurrentRCSyncs, wait.NeverStop)
}
