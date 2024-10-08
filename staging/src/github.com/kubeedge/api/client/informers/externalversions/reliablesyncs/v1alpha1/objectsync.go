/*
Copyright The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	reliablesyncsv1alpha1 "github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	versioned "github.com/kubeedge/api/client/clientset/versioned"
	internalinterfaces "github.com/kubeedge/api/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubeedge/api/client/listers/reliablesyncs/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ObjectSyncInformer provides access to a shared informer and lister for
// ObjectSyncs.
type ObjectSyncInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ObjectSyncLister
}

type objectSyncInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewObjectSyncInformer constructs a new informer for ObjectSync type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewObjectSyncInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredObjectSyncInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredObjectSyncInformer constructs a new informer for ObjectSync type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredObjectSyncInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ReliablesyncsV1alpha1().ObjectSyncs(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ReliablesyncsV1alpha1().ObjectSyncs(namespace).Watch(context.TODO(), options)
			},
		},
		&reliablesyncsv1alpha1.ObjectSync{},
		resyncPeriod,
		indexers,
	)
}

func (f *objectSyncInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredObjectSyncInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *objectSyncInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&reliablesyncsv1alpha1.ObjectSync{}, f.defaultInformer)
}

func (f *objectSyncInformer) Lister() v1alpha1.ObjectSyncLister {
	return v1alpha1.NewObjectSyncLister(f.Informer().GetIndexer())
}
