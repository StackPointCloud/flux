package helm

import (
	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster"
	"github.com/weaveworks/flux/ssh"
)

// Helm implements the Cluster interface, using interactions with
// a Tiller pod to implement the service and manifest methods
type Helm struct {
	// client *khelm.Client
}

// // getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
// func getKubeClient() (*rest.Config, kubernetes.Interface, error) {

// }

// // GetConfig returns a kubernetes client config for a given context.
// // https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration
// // https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration
// func GetConfig(context string) clientcmd.ClientConfig {

// 	kubeconfig = os.Getenv("KUBECONFIG")

// 	// use the current context in kubeconfig
// 	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
// 	}

// 	// create the clientset
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
// 	}

// 	// creates the in-cluster config
// 	// config, err := rest.InClusterConfig()
// 	// if err != nil {
// 	// 	panic(err.Error())
// 	// }
// 	return config, client, nil
// }

// NewHelm creates a Helm instance from ... the necessary arguments
func NewHelm() (cluster.Cluster, error) {

	return NewHelmCLI()
	// config, client, err := getKubeClient(kubeContext)
	// if err != nil {
	// 	return nil, err
	// }

	// tunnel, err := portforwarder.New(settings.TillerNamespace, client, config)
	// if err != nil {
	// 	return nil, err
	// }

	// tillerHost := fmt.Sprintf("localhost:%d", tunnel.Local)
	// debug("Created tunnel using local port: '%d'\n", tunnel.Local)

	// // Set up the gRPC config.
	// debug("SERVER: %q\n", tillerHost)

	// helmClient := khelm.NewClient(khelm.Host(tillerHost))
	// return &Helm{
	// 	client: helmClient,
	// }, nil
}

// AllServices
func (helm *Helm) AllServices(maybeNamespace string) ([]cluster.Service, error) {
	return nil, nil
}

// SomeServices
func (helm *Helm) SomeServices([]flux.ServiceID) ([]cluster.Service, error) {
	return nil, nil
}

// Ping
func (helm *Helm) Ping() error {
	return nil
}

// Export
func (helm *Helm) Export() ([]byte, error) {
	return nil, nil
}

// Sync
func (helm *Helm) Sync(cluster.SyncDef) error {
	return nil
}

// PublicSSHKey
func (helm *Helm) PublicSSHKey(regenerate bool) (ssh.PublicKey, error) {
	return ssh.PublicKey{}, nil
}
