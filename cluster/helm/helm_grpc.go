package helm

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v2"

	// "k8s.io/client-go/kubernetes" // must use 1.5 versions otherwise clash
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/clientcmd"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster"
	"github.com/weaveworks/flux/cluster/helm/proto/hapi/services"
	fkubernetes "github.com/weaveworks/flux/cluster/kubernetes"
	"github.com/weaveworks/flux/ssh"
)

func Debug(messages ...interface{}) {
	l := len(messages)
	for i, m := range messages {
		fmt.Printf("DEBUG %d-%d   %v\n", l, i+1, m)
	}
}

// HelmGRPC implements the Cluster interface,
type HelmGRPC struct {
	tillerClient services.ReleaseServiceClient
	kubeClient   *kubernetes.Clientset
}

func connect(host string, useTLS bool, tlsConfig *tls.Config) (conn *grpc.ClientConn, err error) {
	opts := []grpc.DialOption{
		grpc.WithTimeout(5 * time.Second),
		grpc.WithBlock(),
	}
	switch {
	case useTLS:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	default:
		opts = append(opts, grpc.WithInsecure())
	}
	if conn, err = grpc.Dial(host, opts...); err != nil {
		return nil, err
	}
	return conn, nil
}

func newKubernetesClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// in-cluster
		config, err = rest.InClusterConfig()
	} else {
		// assume external
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// NewHelmGRPC creates a HelmGRPC instance ...
func NewHelmGRPC() (*HelmGRPC, error) {

	tillerHost := os.Getenv("TILLER_HOST")
	if tillerHost == "" {
		tillerHost = "tiller-deploy:44134" // expected in-cluster value. For testing, kubectl port-forward allows "localhost:44134"
	}

	connection, err := connect(tillerHost, false, nil)
	if err != nil {
		return nil, err
	}
	client := services.NewReleaseServiceClient(connection)

	kubeClient, err := newKubernetesClient()
	if err != nil {
		return nil, err
	}

	// kubernetes Cluster impl starts a loop asynchronously to process actions
	// sent from from Sync(). Seems legit.

	return &HelmGRPC{
		tillerClient: client,
		kubeClient:   kubeClient,
	}, nil

}

func newHelmContext() context.Context {
	md := metadata.Pairs("x-helm-api-client", "v2.5.0")
	return metadata.NewContext(context.TODO(), md)
}

// AllServices
func (helm *HelmGRPC) AllServices(maybeNamespace string) ([]cluster.Service, error) {
	ctx := newHelmContext()

	var svcs []cluster.Service

	req := &services.ListReleasesRequest{}
	listReleasesClient, err := helm.tillerClient.ListReleases(ctx, req)
	if err != nil {
		return nil, err
	}
	responses, err := listReleasesClient.Recv()
	if err != nil {
		return nil, err
	}
	releases := responses.GetReleases()
	for _, r := range releases {

		// fmt.Println(r.GetConfig().GetRaw())  // values.yaml
		// fmt.Println(r.GetManifest())  // all manifests
		// fmt.Println(r.GetInfo()) // useless text for humans
		// fmt.Println(r.GetChart().GetValues().GetRaw())

		// (a) pull image references from the values file
		images, err := getYamlImageRefsFromValues(r.GetChart().GetValues().GetRaw())
		if err != nil {
			return nil, err
		}
		for _, i := range images {
			Debug(i.repository, i.tag)
		}

		// (b) pull image references from the manifests
		containers, err := getYamlImageRefsFromManifest(r.GetManifest())
		if err != nil {
			return nil, err
		}
		for _, c := range containers {
			Debug(c.Name, c.Image)
		}

		// (a) vs. (b)
		// (a) values file => clear intent by developer that this image is to be modified
		// (b) manifests => get complete set of *all* images

		// prefer (a).  But need to associate it with particular service ...
		// so ... look for matching container in manifests? find the associated service via
		// labels?  Or do we just say the the flux cluster.Service == the helm chart.

		// alternative -- require that the service have the same name as the helm release.
		// seems alright, service should be named with template-fullname

		serviceName := fmt.Sprintf("%s-%s", r.GetName(), r.GetChart().GetMetadata().GetName())

		// no must use kubernetes client to get status
		// serviceIP, err := findHelmServiceIP(serviceName, r.GetManifest())
		// if err != nil {
		// 	return nil, err
		// }

		var clusterIP string
		k8sService, err := helm.kubeClient.Core().Services(r.GetNamespace()).Get(serviceName)
		if err == nil {
			clusterIP = k8sService.Spec.ClusterIP
		} else {
			Debug("error finding k8s service clusterIP %v", err)
		}

		service := cluster.Service{
			ID:       flux.MakeServiceID(r.GetNamespace(), serviceName),
			IP:       clusterIP, // service.Spec.ClusterIP
			Metadata: map[string]string{},
			Status:   fkubernetes.StatusReady,
			Containers: cluster.ContainersOrExcuse{
				Containers: containers,
			},
		}

		Debug(service)

		svcs = append(svcs, service)

	}
	return svcs, nil
}

// SomeServices filters the services and returns only those with matching ids
func (helm *HelmGRPC) SomeServices(serviceIDs []flux.ServiceID) ([]cluster.Service, error) {

	lookingFor := map[flux.ServiceID]bool{}
	for _, id := range serviceIDs {
		Debug(id)
		lookingFor[id] = true
	}

	services := []cluster.Service{}
	clusterServices, err := helm.AllServices("")
	if err != nil {
		return services, err
	}
	for _, s := range clusterServices {
		Debug(s.ID)
		if lookingFor[s.ID] {
			services = append(services, s)
		}

	}
	return services, nil
}

// Ping ensures that there's a connection to the tiller server
func (helm *HelmGRPC) Ping() error {
	ctx := newHelmContext()

	version, err := helm.tillerClient.GetVersion(ctx, &services.GetVersionRequest{})
	Debug(version.GetVersion().String())
	if err != nil {
		return err
	}
	return nil
}

// Export.  Probably want to reconstruct the whole chart, who calls this?
func (helm *HelmGRPC) Export() ([]byte, error) {
	return nil, fmt.Errorf("Not implemented")
}

// Sync processes a sequence of cluster.SyncAction
// SyncAction.Delete should be a serialized serivces.UninstallReleaseRequest
// SyncAction.Apply should be a serialzed services.UpdateReleaseRequest or services.InstallReleaseRequest ...
// which we deserialize here and apply.  Or somehow, here, we construct those objects from the []byte
// that are passed in.
func (helm *HelmGRPC) Sync(syncRequest cluster.SyncDef) error {
	for _, action := range syncRequest.Actions {
		Debug(action.ResourceID)
	}

	return fmt.Errorf("Not implemented")
}

// PublicSSHKey
func (helm *HelmGRPC) PublicSSHKey(regenerate bool) (ssh.PublicKey, error) {
	return ssh.PublicKey{}, fmt.Errorf("Not implemented")
}

type imageRef struct {
	repository string `yaml:"repository"`
	tag        string `yaml:"tag"`
}

// requires that the specification be in the values.yaml file in the format
// somename:
//   image:
//      repository: repo/developer/name
//      tag: v1.2.3
func getYamlImageRefsFromValues(valuesData string) ([]imageRef, error) {
	values, err := unmarshalYaml([]byte(valuesData))
	if err != nil {
		return []imageRef{}, err
	}

	return retrieveImageRefs(values), nil
}

// no good, spec.clusterIP spec is not in manifest -- must query kubernetes
func findHelmServiceIP(releaseName, manifestData string) (string, error) {
	documents := strings.Split(manifestData, "---")
	for _, doc := range documents {
		values, err := unmarshalYaml([]byte(doc))
		if err != nil {
			return "", err
		}
		if values["kind"] == "Service" {
			if metadata, ok := values["metadata"].(map[string]interface{}); ok {
				if metadata["name"] == releaseName {
					if spec, ok := values["spec"].(map[string]interface{}); ok {
						// fmt.Printf("%s   %v\n", releaseName, spec)
						if spec["clusterIP"] != nil {
							return spec["clusterIP"].(string), nil
						}
						return "", nil
					}
				}
			}
		}

	}
	return "", fmt.Errorf("service <%s> not found in helm manifests", releaseName)

}

func getYamlImageRefsFromManifest(manifestData string) ([]cluster.Container, error) {
	var containers []cluster.Container
	// the manifestData is multiple YAML documents, separated wiuth "---"
	documents := strings.Split(manifestData, "---")
	for _, doc := range documents {
		values, err := unmarshalYaml([]byte(doc))
		if err != nil {
			return []cluster.Container{}, err
		}
		containers = append(containers, retrieveContainers(values)...)
	}
	return containers, nil
}

func retrieveContainers(yamlContent map[string]interface{}) []cluster.Container {
	containers := []cluster.Container{}
	// "containers" should be an array of elements
	if yamlContent["containers"] != nil {
		if yamlContainers, ok := yamlContent["containers"].([]interface{}); ok {

			for _, yc := range yamlContainers {
				c := yc.(map[string]interface{})
				name := c["name"].(string)
				image := c["image"].(string)
				// fmt.Printf("  -->  %v\n      --->%s %s\n", c, name, image)

				if name != "" && image != "" {
					containers = append(containers, cluster.Container{
						Image: image,
						Name:  name,
					})
				}
			}
		}
	}
	for _, v := range yamlContent {
		// fmt.Printf("value for %s is %T\n", k, v)
		if subYamlData, ok := v.(map[string]interface{}); ok {
			containers = append(containers, retrieveContainers(subYamlData)...)
		}
	}
	return containers
}

func retrieveImageRefs(yamlContent map[string]interface{}) []imageRef {
	images := []imageRef{}
	if yamlContent["image"] != nil {
		if yamlImage, ok := yamlContent["image"].(map[string]interface{}); ok {
			repo := yamlImage["repository"].(string)
			tag := yamlImage["tag"].(string)
			if repo != "" && tag != "" {
				images = append(images, imageRef{
					repository: repo,
					tag:        tag,
				})
			}
		}
	}
	for _, v := range yamlContent {
		if subYamlData, ok := v.(map[string]interface{}); ok {
			images = append(images, retrieveImageRefs(subYamlData)...)
		}
	}
	return images
}

func unmarshalYaml(data []byte) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(data), &res); err != nil {
		return res, err
	}
	for k, v := range res {
		res[k] = cleanupMapValue(v)
	}
	return res, nil
}

func cleanupInterfaceArray(in []interface{}) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = cleanupMapValue(v)
	}
	return res
}

func cleanupInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range in {
		res[fmt.Sprintf("%v", k)] = cleanupMapValue(v)
	}
	return res
}

func cleanupMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanupInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanupInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
