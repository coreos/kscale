package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/controller"
	replicationcontroller "k8s.io/kubernetes/pkg/controller/replication"
	"k8s.io/kubernetes/pkg/util/wait"
)

const lookupCacheSizeForRC = 4096 // query controller by pod cache

func main() {
	var apisrvAddr string
	var concurrentRCSyncs int
	var pprofPort int
	var dumpdir string
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	flag.StringVar(&dumpdir, "dumpdir", "dump", "dump dir")
	flag.IntVar(&concurrentRCSyncs, "p", 5, "Concurrent RC goroutines")
	flag.IntVar(&pprofPort, "pprof-port", 6060, "local http handler")
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil))
	}()

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
