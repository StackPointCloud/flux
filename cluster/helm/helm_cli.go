package helm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster"
	"github.com/weaveworks/flux/ssh"
)

// HelmCLI implements the Cluster interface, using an instance of the Helm binary
type HelmCLI struct {
	Repository string
	Executable string
	HelmHome   string
	Kubeconfig string
}

// NewHelmCLI creates a HelmCLI instance using environment variables
// we'll assert that everything has been configured in the external
// environment, including helm, kubectl and a kubeconfig file.
// Note this can be made to work fine on a pod inside a cluster
// by constructing a kubeconfig file from the serviceaccount token
// /var/run/secrets/kubernetes.io/serviceaccount/token and cert
// /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
func NewHelmCLI() (*HelmCLI, error) {

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, fmt.Errorf("helm not available, %v ", err)
	}

	home := os.Getenv("HELM_HOME")
	if home == "" {
		home = os.Getenv("HOME") + "/.helm"
	}
	err = os.MkdirAll(home, 0755)
	if err != nil {
		return nil, fmt.Errorf("problem accessing %s, %v ", home, err)
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}
	_, err = os.Stat(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("problem accessing kubeconfig %s, %v ", kubeconfig, err)
	}

	return &HelmCLI{
		Executable: helmPath,
		HelmHome:   home,
		Kubeconfig: kubeconfig,
	}, nil
}

// execute runs the helm cli, returning stdout, stderr, and any error
func (client *HelmCLI) execute(arguments []string) ([]byte, []byte, error) {

	helmEnv := os.Environ()
	helmEnv = append(helmEnv, fmt.Sprintf("HELM_HOME=%s", client.HelmHome))
	helmEnv = append(helmEnv, fmt.Sprintf("KUBECONFIG=%s", client.Kubeconfig))

	cmdOut := &bytes.Buffer{}
	cmdErr := &bytes.Buffer{}
	fmt.Printf("DEBUG, %s %v\n", "helm", arguments)
	cmd := exec.Cmd{
		Path:   client.Executable,
		Env:    helmEnv,
		Args:   append([]string{client.Executable}, arguments...),
		Stdout: cmdOut,
		Stderr: cmdErr,
	}
	err := cmd.Run()
	return cmdOut.Bytes(), cmdErr.Bytes(), err
}

// AllServices
func (helm *HelmCLI) AllServices(maybeNamespace string) ([]cluster.Service, error) {
	releases, err := helm.listReleases(maybeNamespace)
	if err != nil {
		return []cluster.Service{}, err
	}
	services := make([]cluster.Service, len(releases))
	for i, release := range releases {
		fmt.Printf("DEBUG: %v\n", release)
		services[i] = cluster.Service{
			ID:       flux.MakeServiceID(release.namespace, release.name),
			IP:       "0.0.0.0",
			Metadata: nil,
			Status:   release.status,
			Containers: cluster.ContainersOrExcuse{
				Excuse: "sorry",
			},
		}
	}
	return services, nil
}

// SomeServices
func (helm *HelmCLI) SomeServices([]flux.ServiceID) ([]cluster.Service, error) {
	return nil, fmt.Errorf("Not implemented")
}

// Ping
func (client *HelmCLI) Ping() error {
	version, err := client.serverVersion()
	fmt.Printf("DEBUG, version %s\n", version)
	return err
}

// Export
func (helm *HelmCLI) Export() ([]byte, error) {
	return nil, fmt.Errorf("Not implemented")
}

// Sync
func (helm *HelmCLI) Sync(cluster.SyncDef) error {
	return fmt.Errorf("Not implemented")
}

// PublicSSHKey
func (helm *HelmCLI) PublicSSHKey(regenerate bool) (ssh.PublicKey, error) {
	return ssh.PublicKey{}, nil
}

func (client *HelmCLI) serverVersion() (string, error) {
	stdout, _, err := client.execute([]string{"version"})
	if err != nil {
		return "", fmt.Errorf("Version failed, %v", err)
	}
	matcher, _ := regexp.Compile("Server: &version.Version{SemVer:\"(v[.0-9]+)")
	versionFound := matcher.FindStringSubmatch(string(stdout))
	if versionFound == nil || len(versionFound) < 2 {
		return "", fmt.Errorf("Server version not found in response text <<%s>>", stdout)
	}
	return versionFound[1], nil
}

type releaseSummary struct {
	name       string
	revision   int64
	deployTime time.Time
	status     string
	chart      string
	version    string
	namespace  string
}

// func (summary releaseSummary) asService() cluster.Service {
// 	svc := cluster.Service{}
// }

func parseReleaseResponseLine(line string) (releaseSummary, error) {
	// line := "promop	1       	Tue Jul 11 23:05:37 2017	 DEPLOYED	prometheus-operator-0.0.6	monitoring"
	fmt.Printf("DEBUG, parsing \"%s\"\n", line)
	words := strings.Fields(line)
	if len(words) != 10 {
		return releaseSummary{}, fmt.Errorf("Unparseable line: << %s >>", line)
	}
	deployTime, err := time.Parse("Mon Jan 02 15:04:05 2006", strings.Join(words[2:7], " "))
	if err != nil {
		fmt.Printf("DEBUG: deployTime %v", err) // shrug
	}
	rev, err := strconv.ParseInt(words[1], 10, 32)
	if err != nil {
		fmt.Printf("DEBUG: revision %v", err) // shrug
	}

	pieces := strings.Split(words[8], "-")
	last := len(pieces) - 1
	var version string
	if last > 0 {
		version = pieces[last]
	}
	chart := strings.Join(pieces[0:last], "-")

	return releaseSummary{
		name:       words[0],
		revision:   rev,
		deployTime: deployTime,
		status:     words[7],
		chart:      chart,
		version:    version,
		namespace:  words[9],
	}, nil
}

func (client *HelmCLI) listReleases(namespace string) ([]releaseSummary, error) {
	var arguments []string
	if namespace == "" {
		arguments = []string{"list"}
	} else {
		arguments = []string{"list", "--namespace", namespace}
	}
	stdout, _, err := client.execute(arguments)
	if err != nil {
		return []releaseSummary{}, fmt.Errorf("listReleases failed, %v", err)
	}
	responseLines := strings.Split(string(stdout), "\n")
	count := len(responseLines)
	var summary []releaseSummary
	for i := 1; i < count; i++ {
		s, err := parseReleaseResponseLine(responseLines[i])
		if err == nil {
			summary = append(summary, s)
		} else {
			fmt.Printf("DEBUG parsing release line failed, %v\n", err)
		}
	}
	return summary, nil
}
