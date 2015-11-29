Scheduler Benchmark
======

Background
------
We want to benchmark the scheduler's performance.
We need to make sure scheduler is not the throttler in the entire Kubernetes system.

Our target scale is 1000 nodes and 30K pods to schedule.

Technical Approach
------
Here we focus on the scheduler per se.
The overall flow of how scheduler works:

1. Scheduler [watches pod resource](https://github.com/kubernetes/kubernetes/blob/cadc24e9fd7f2bccc972df4d67985aa33a4cd823/plugin/pkg/scheduler/factory/factory.go#L192) changes and [pushes it into PodQueue](https://github.com/kubernetes/kubernetes/blob/cadc24e9fd7f2bccc972df4d67985aa33a4cd823/plugin/pkg/scheduler/factory/factory.go#L229).
2. Scheduler [polls from PodQueue](https://github.com/kubernetes/kubernetes/blob/3d5207fd733fdc2cdd3dc17ed0bb254d406a633d/plugin/pkg/scheduler/scheduler.go#L114) and make binding/scheduling decision.
3. Scheduler writes the decision into [binding resource](https://github.com/kubernetes/kubernetes/blob/cadc24e9fd7f2bccc972df4d67985aa33a4cd823/plugin/pkg/scheduler/factory/factory.go#L360).

Ultimately, we want to benchmark Step 2 which is the real performance of scheduler.
However, it's not very easy to mock the interface of scheduler at the moment.
As a compromise, we can take advantage of the isolated design of API server and scheduler: we can have a mock api server, gives out need-to-schedule pods, fakes some number of nodes and information, and catches any pod-node bindings. The following diagram shows the plan.

![](scheduler.png)
