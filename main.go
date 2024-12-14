package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/kstatus/watcher"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func other() error {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil)
	cfg, err := clientCfg.ClientConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return err
	}
	restMapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	if err != nil {
		return err
	}
	sw := watcher.NewDefaultStatusWatcher(dynamicClient, restMapper)
	fmt.Println(clientset)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	objs := []object.ObjMetadata{
		{
			Namespace: "podinfo",
			Name:      "podinfo",
			GroupKind: schema.GroupKind{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}
	err = WaitForReady(ctx, sw, objs)
  if err != nil {
    return fmt.Errorf("error waiting for ready: %w", err)
  }
	fmt.Println("resources are ready")
  return nil
}

func main() {
	err := other()
	if err != nil {
		fmt.Println("error is", err.Error())
	}
}

// WaitForReady waits for all of the objects to reach a ready state.
func WaitForReady(ctx context.Context, sw watcher.StatusWatcher, objs []object.ObjMetadata) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	eventCh := sw.Watch(cancelCtx, objs, watcher.Options{})
	statusCollector := collector.NewResourceStatusCollector(objs)
	done := statusCollector.ListenWithObserver(eventCh, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, _ event.Event) {
			rss := []*event.ResourceStatus{}
			for _, rs := range statusCollector.ResourceStatuses {
				if rs == nil {
					continue
				}
				rss = append(rss, rs)
			}
			desired := status.CurrentStatus
			if aggregator.AggregateStatus(rss, desired) == desired {
				cancel()
				return
			}
		}),
	)
	<-done

	if statusCollector.Error != nil {
		return statusCollector.Error
	}

	// Only check parent context error, otherwise we would error when desired status is achieved.
	if ctx.Err() != nil {
		errs := []error{}
		for _, id := range objs {
			rs := statusCollector.ResourceStatuses[id]
			fmt.Println(rs.Status)
			switch rs.Status {
			case status.CurrentStatus:
				fmt.Println("ready")
			case status.NotFoundStatus:
				errs = append(errs, fmt.Errorf("%s: %s not found", rs.Identifier.Name, rs.Identifier.GroupKind.Kind))
			case status.UnknownStatus:
				errs = append(errs, fmt.Errorf("%s: %s unknown status", rs.Identifier.Name, rs.Identifier.GroupKind.Kind))
			default:
				errs = append(errs, fmt.Errorf("%s: %s not ready", rs.Identifier.Name, rs.Identifier.GroupKind.Kind))
			}
		}
		errs = append(errs, ctx.Err())
		return errors.Join(errs...)
	}

	return nil
}
