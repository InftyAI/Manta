/*
Copyright 2024.

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

package wrapper

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodWrapper struct {
	corev1.Pod
}

func MakePod(name, namespace string) *PodWrapper {
	return &PodWrapper{
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "default",
						Image: "foo:bar",
					},
				},
			},
		},
	}
}

func (w *PodWrapper) Obj() *corev1.Pod {
	return &w.Pod
}

func (w *PodWrapper) Label(k, v string) *PodWrapper {
	if w.Labels == nil {
		w.Labels = map[string]string{}
	}
	w.Labels[k] = v
	return w
}

func (w *PodWrapper) InitContainer(name string) *PodWrapper {
	c := corev1.Container{Name: name}
	w.Spec.InitContainers = append(w.Spec.InitContainers, c)
	return w
}

func (w *PodWrapper) InitContainerImage(name string, image string) *PodWrapper {
	for i, c := range w.Spec.InitContainers {
		if c.Name == name {
			w.Spec.InitContainers[i].Image = image
		}
	}
	return w
}

func (w *PodWrapper) InitContainerImagePolicy(name string, pullPolicy string) *PodWrapper {
	for i, c := range w.Spec.InitContainers {
		if c.Name == name {
			w.Spec.InitContainers[i].ImagePullPolicy = corev1.PullPolicy(pullPolicy)
		}
	}
	return w
}

func (w *PodWrapper) InitContainerCommands(name string, commands ...string) *PodWrapper {
	for i, c := range w.Spec.InitContainers {
		if c.Name == name {
			w.Spec.InitContainers[i].Command = commands
		}
	}
	return w
}

func (w *PodWrapper) InitContainerPort(name string, portName string, containerPort int32, protocol string) *PodWrapper {
	port := corev1.ContainerPort{
		Name:          portName,
		ContainerPort: containerPort,
		Protocol:      corev1.Protocol(protocol),
	}

	for i, c := range w.Spec.InitContainers {
		if c.Name == name {
			w.Spec.InitContainers[i].Ports = append(w.Spec.InitContainers[i].Ports, port)
		}
	}
	return w
}
