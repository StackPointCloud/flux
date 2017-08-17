## Reassessment of the flux-helm integration

After partial implementation of pieces of the work, it is necessary to take a step back and 
reassess the general work plan and lay out the larger scope.

Flux is a CI system which monitors 
  1. an application installed in a kubernetes cluster
  2. the kubernetes manifests used to deploy that application, stored in a github repository
  3. the containers running that application, whose images are stored in a container registry

The concept is to replace the kubernetes manifests and the direct interactions with
the kubernetes api server, with helm charts and interactions with the tiller server.

### Integration with helm

Flux is currently designed to work with a kubernetes cluster, talking to the kubernetes apiserver.  
The interactions are represented by a couple of interfaces, so it appears that by re-implementing
those interfaces with a helm backend, we can easily provide a new viable approach for developers
using helm.   

#### Interfaces to implement

The interfaces that flux uses to represent the application and its configuration are
   1. Cluster
   2. Manifest

The cluster interface in the interaction with the currently running system.

The manifest interface represents a particular service (or application) and also represents
the interactions with the source code repository.

#### Choices for implementation

 - integrate helm directly into the code
 - helm client cli
 - grpc connections

[1] Each of these three options has problems.  Integrating the helm codebase directly and calling 
using the internal helm client is problematic because flux and helm use two different 
golang dependency management tools (gvt vs godep) and flux is using the kubernetest v1.5 
client libraries.  Updating the client libraries and reworking dependencies for flux is a 
significant effort. (Note that a comparable project, https://github.com/rusenask/keel does 
not have these issues, and includes helm libraries directly).  

[2] Alternatively, flux could use the helm cli, invoking helm as an external process and parsing the
stdout for results.  This is pretty doable, but requires writing parsing the human-oriented
output messages, which is at least tedious and the implicit api is very much not guaranteeed to 
be stable.  

[3] Using grpc to talk to the helm api is how helm internally does it.  The go code is generated 
from grpc definitions, and we can crudely copy the generated libraries into the flux codebase 
and use them as an interface. In this case, the problem reduces to finding a match between the
tiller grpc interfaces and the desired flux cluster and manifest interfaces.  The grpc interfaces
provide access to the helm chart contents as byte arrays, and it's possible to reconstruct and parse
the yaml contents to extract relevant information. 

Unfortunately, because the helm interfaces don't provide all the details necessary, the flux-helm 
client has to interact with both the tiller api and the kubernetes api and combine the two.
One place where this is necessary is finding the service ip in the cluster, which is not
exposed via helm at all.  The only place where helm generally exposes this sort of information
is in an optional notes.txt file, written out to the console when the chart is installed.

Most of the cluster interface is now implemented against helm/grpc. Implementing the manifest 
interface and interacting with source code repositories to detect changes is work to be done.

### Utilization of the new interfaces

The cluster and manifest interfaces are used by the fluxd commnand application.  The construction
of the objects are heavily dependent on the kubernetes implementation, and the entire fluxd/main.go
command must be completely refactored in order to switch between the current kubernetes-based interfaces
and the new the helm-based interfaces.

The fluxd command application constructs a daemon object to use the interfaces, and it's possible that 
the daemon package is properly agnostic about the nature of the implementation.  If so, the
refactoring is limited to the command invocation and all of the commandline options that 
are typically used.

Refactoring should be done by engineers already well-familiar with all of the flux components. 

### Minimum viable product

THe initial goal was to simply have a MVP using helm.  There are a lot of open questions however, since
helm is itself fairly complex.  Helm charts can be used to deploy not just backend application containers
with frontend services, but also external ingress access configurations, backend storage settings, 
backend secret management tools and much more.  The MVP definition should be carefully pinned down.  
We'd want something that demonstrates the basic features of helm in an already usable way, doesn't get 
caught up in helm features that flux will never use, but is clearly extensible toward features that 
flux uses elsewhere.






