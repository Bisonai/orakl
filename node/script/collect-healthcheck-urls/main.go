package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type HealthCheckUrl struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

func handleError(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func getHealthPort(service *corev1.Service) int32 {
	for _, port := range service.Spec.Ports {
		if port.Name == "healthcheck" {
			return port.Port
		}
	}
	if len(service.Spec.Ports) > 0 {
		return service.Spec.Ports[0].Port
	}
	return 0
}

func getHealthCheckHttp(ctx context.Context, clientset *kubernetes.Clientset, namespace string) []HealthCheckUrl {
	var result []HealthCheckUrl

	services, err := clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	handleError(err)

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	handleError(err)

	for _, service := range services.Items {
		healthPort := getHealthPort(&service)
		servicePods := getRelatedPods(pods, &service)

		for _, pod := range servicePods {
			for _, container := range pod.Spec.Containers {
				path := ""
				if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
					path = container.ReadinessProbe.HTTPGet.Path
					url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", service.Name, service.Namespace, healthPort, path)
					result = append(result, HealthCheckUrl{Name: service.Name, Url: url})
				}

			}
		}
	}

	return result
}

func getHealthCheckRedis(ctx context.Context, clientset *kubernetes.Clientset, namespace string) []HealthCheckUrl {
	var result []HealthCheckUrl

	services, err := clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	handleError(err)

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	handleError(err)

	for _, service := range services.Items {
		healthPort := getHealthPort(&service)
		servicePods := getRelatedPods(pods, &service)

		for _, pod := range servicePods {
			for _, container := range pod.Spec.Containers {
				if container.ReadinessProbe == nil && container.ReadinessProbe.HTTPGet == nil {
					continue
				}
				url := fmt.Sprintf("redis://%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, healthPort)
				result = append(result, HealthCheckUrl{Name: service.Name, Url: url})
			}
		}
	}
	return result

}

func getSavePath(chain string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(wd, "pkg/checker/health", chain+"_healthcheck.json")
	return configPath, nil
}

func getRelatedPods(pods *corev1.PodList, service *corev1.Service) []*corev1.Pod {
	var relatedPods []*corev1.Pod
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, service.Name) {
			relatedPods = append(relatedPods, &pod)
		}
	}
	return relatedPods
}

/*
to run the script, kubectl should be set up
*/
func main() {
	ctx := context.Background()
	user := os.Getenv("USER")
	chain := flag.String("chain", "baobab", "the chain to use (baobab or cypress)")
	kubeconfig := flag.String("kubeconfig", "/Users/"+user+"/.kube/config", "location to your kubeconfig file")
	contextName := flag.String("context", "orakl-baobab-admin@bisonai", "the context to use")

	flag.Parse()

	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig}
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: *contextName}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides)

	clientConfig, err := config.ClientConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		panic(err)
	}

	var result []HealthCheckUrl

	result = append(result, getHealthCheckHttp(ctx, clientset, "orakl")...)

	result = append(result, getHealthCheckRedis(ctx, clientset, "redis")...)

	savePath, err := getSavePath(*chain)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(savePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(result)
	if err != nil {
		panic(err)
	}

	fmt.Println("Health check urls are saved to", savePath)
}
