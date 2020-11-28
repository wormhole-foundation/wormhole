package e2e

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	TiltDefaultNamespace = metav1.NamespaceDefault // hardcoded Tilt assumption
)

func getk8sClient() *kubernetes.Clientset {
	config, err := getk8sConfig()

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorln(err)
	}
	return c
}

func getk8sConfig() (*rest.Config, error) {
	// Load local default kubeconfig.
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.ClientConfigLoader(rules), nil).ClientConfig()
}

func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func waitForPods(ctx context.Context, c *kubernetes.Clientset, want []string) {
	found := make(map[string]bool)
	ctx, cancel := context.WithCancel(ctx)

	watchlist := cache.NewListWatchFromClient(
		c.CoreV1().RESTClient(),
		"pods",
		TiltDefaultNamespace,
		fields.Everything(),
	)

	handle := func(pod *corev1.Pod) {
		ready := hasPodReadyCondition(pod.Status.Conditions)
		log.Printf("pod added/changed: %s is %s, ready: %v", pod.Name, pod.Status.Phase, ready)

		if ready {
			found[pod.Name] = true
		}

		missing := 0
		for _, v := range want {
			if found[v] == false {
				missing += 1
			}
		}

		if missing == 0 {
			cancel()
		}
	}

	_, controller := cache.NewInformer(
		watchlist,
		&corev1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { handle(obj.(*corev1.Pod)) },
			UpdateFunc: func(oldObj, newObj interface{}) { handle(newObj.(*corev1.Pod)) },
		},
	)

	controller.Run(ctx.Done())
}

func executeCommandInPod(ctx context.Context, c *kubernetes.Clientset, podName string, container string, cmd []string) ([]byte, error) {
	p, err := c.CoreV1().Pods(TiltDefaultNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", p, err)
	}

	req := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(TiltDefaultNamespace).
		SubResource("exec")

	req = req.VersionedParams(&corev1.PodExecOptions{
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
		Container: container,
		Command:   cmd,
	}, scheme.ParameterCodec)

	config, err := getk8sConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to init executor: %w", err)
	}

	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    false,
	})

	log.Printf("command: %s", strings.Join(cmd, " "))
	if execErr.Len() > 0 {
		log.Printf("stderr: %s", execErr.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute remote command: %w", err)
	}

	return execOut.Bytes(), nil
}
