package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"

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
	// var dumpdir string
	flag.StringVar(&apisrvAddr, "addr", "localhost:8080", "APIServer addr")
	// flag.StringVar(&dumpdir, "dumpdir", "dump", "dump dir")
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

	rcm := replicationcontroller.NewReplicationManagerFromClient(
		clientset.NewForConfigOrDie(restclient.AddUserAgent(kubeconfig, "replication-controller")),
		controller.NoResyncPeriodFunc,
		replicationcontroller.BurstReplicas,
		lookupCacheSizeForRC,
	)
	go rcm.Run(concurrentRCSyncs, wait.NeverStop)

	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, os.Interrupt, os.Kill)
	fmt.Println("waiting for signal")
	sig := <-notifier
	fmt.Printf("sig: %v\n", sig)
	runtime.GC()
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	fmt.Printf("alloc: %d, sys: %d, idle: %d, inuse: %d\n", st.HeapAlloc, st.HeapSys, st.HeapIdle, st.HeapInuse)
}
