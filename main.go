package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/kstatus/watcher"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func watch() error {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil)
	cfg, err := clientCfg.ClientConfig()
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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
	fmt.Println("resources are ready after watch")
	return nil
}

func main() {
	err := watch()
	if err != nil {
		fmt.Println("error is", err.Error())
	}
	err = poll()
	if err != nil {
		fmt.Println("error is", err.Error())
	}
}

func poll() error {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil)
	cfg, err := clientCfg.ClientConfig()
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
	c, err := client.New(cfg, client.Options{Mapper: restMapper})
	if err != nil {
		return err
	}
	poll := polling.NewStatusPoller(c, c.RESTMapper(), polling.Options{})
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = WaitForReadyPoll(ctx, poll, objs)
	if err != nil {
		return fmt.Errorf("error waiting for ready: %w", err)
	}
	fmt.Println("resources are ready after poll")
	return nil
}

// WaitForReady waits for all of the objects to reach a ready state.
func WaitForReadyPoll(ctx context.Context, sp *polling.StatusPoller, objs []object.ObjMetadata) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := polling.PollOptions{PollInterval: 1 * time.Second}
	eventsChan := sp.Poll(cancelCtx, objs, opts)
	statusCollector := collector.NewResourceStatusCollector(objs)
	done := statusCollector.ListenWithObserver(eventsChan, collector.ObserverFunc(
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
