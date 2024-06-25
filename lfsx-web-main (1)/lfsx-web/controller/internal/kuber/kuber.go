// kuber handles requests to the OpenShift / kubernetes
// cluster
package kuber

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"gitea.hama.de/LFS/lfsx-web/controller/templates"
	"k8s.io/apimachinery/pkg/runtime"
	schemar "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	schema "k8s.io/client-go/kubernetes/scheme"
	cr "sigs.k8s.io/controller-runtime"
)

// Kuber is responsible for handling
// request to the Openshift / kubernetes cluster.
//
// It's named Kuber because it sounds cool and isn't
// conflicting with the kubernetes packages
type Kuber struct {

	// The default namespace to operate in
	Namespace string

	// Kubernetes client used for the API requests
	Client *kubernetes.Clientset

	// App configuration
	appConfig *models.AppConfig
}

// NewKubernetes creates a new kubernetes client for speaking
// with the cluster
func NewKuber(appConfig *models.AppConfig) (*Kuber, error) {

	// Get configuration for speaking with the API
	config, err := cr.GetConfig()
	if err != nil {
		return nil, err
	}

	// Create client used for the API requests
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get the namespace
	namespace, err := getNamespace()
	if err != nil {
		return nil, err
	}

	return &Kuber{
		Namespace: namespace,
		Client:    clientset,
		appConfig: appConfig,
	}, nil
}

// getNamespace returns the currently set namespace
// from the environment variable "KUBERNETES_NAMESPACE"
// or when running inside kubernetes from the service accounts
// namespace
func getNamespace() (string, error) {

	// Lookup from environment variable
	envNameSpaceKey := "KUBERNETES_NAMESPACE"
	if nsEnv, ok := os.LookupEnv(envNameSpaceKey); ok {
		return nsEnv, nil
	}

	// Lookup from service account
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if nsSa := strings.TrimSpace(string(data)); len(nsSa) > 0 {
			return nsSa, nil
		}
	}

	// Return an error because namespace is needed
	return "", fmt.Errorf("unable to get namespace to operate in. Set the env variable %q to provide this information", envNameSpaceKey)
}

// getRessourceFromFile reads an k8 ressource from the relative path of the "template"
// folder and returns an generic object that you can cast base on "GroupVersionKind" to
// a concrete ressource
func (k *Kuber) getRessourceFromFile(path string, templateData any) (runtime.Object, *schemar.GroupVersionKind, error) {

	// Parse template
	template, err := template.ParseFS(templates.TemplateFiles, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse template for pod creation: %s", err)
	}

	// Read template into buffer
	var tmplBuf bytes.Buffer
	if err = template.Execute(&tmplBuf, templateData); err != nil {
		return nil, nil, fmt.Errorf("failed to execute template: %s", err)
	}

	// Log template
	logger.Trc("Parsed template:\n%s", tmplBuf)

	// Decode with universal deserializer
	decode := schema.Codecs.UniversalDeserializer().Decode
	return decode(tmplBuf.Bytes(), nil, nil)
}
