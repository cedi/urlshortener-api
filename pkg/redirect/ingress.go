package redirect

import (
	"fmt"
	"strings"

	"github.com/cedi/urlshortener-api/api/v1alpha1"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewRedirectIngress takes an existing ingress and updates it, or creates an entirely new *networkingv1.Ingress object
func NewRedirectIngress(ing *networkingv1.Ingress, redirect *v1alpha1.Redirect) *networkingv1.Ingress {
	pathTypePrefix := networkingv1.PathTypePrefix

	if ing == nil {
		ing = &networkingv1.Ingress{}
	}

	ing.ObjectMeta = metav1.ObjectMeta{
		Name:      redirect.Name,
		Namespace: redirect.Namespace,
		Labels:    GetLabelsForRedirect(redirect.Name),
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target":          "/",
			"nginx.ingress.kubernetes.io/permanent-redirect":      fmt.Sprintf("http://%s$request_uri", redirect.Spec.Target),
			"nginx.ingress.kubernetes.io/permanent-redirect-code": fmt.Sprintf("%d", redirect.Spec.Code),
		},
	}

	ing.Spec = networkingv1.IngressSpec{
		IngressClassName: &redirect.Spec.IngressClassName,
		Rules: []networkingv1.IngressRule{
			{
				Host: redirect.Spec.Source,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "http-svc",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if redirect.Spec.TLS.Enable == true {
		ing.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{redirect.Spec.Source},
				SecretName: fmt.Sprintf("%s-redirect-secret", strings.ReplaceAll(redirect.Spec.Source, ".", "-")),
			},
		}

		// Add additional annotations based from our TLS spec
		for annotationKey, annotationValue := range redirect.Spec.TLS.Annotations {
			ing.ObjectMeta.Annotations[annotationKey] = annotationValue
		}
	}

	return ing
}

// GetLabelsForRedirect returns the labels for selecting the resources
// belonging to the given redirect CRD name.
func GetLabelsForRedirect(name string) map[string]string {
	return map[string]string{"app": "urlshortener-api", "redirect": name}
}

// GetIngressNames returns a []string from a []networkingv1.Ingress object
// containing only the networkingv1.Ingress.ObjectMeta.Name of the input
func GetIngressNames(ingresses []networkingv1.Ingress) []string {
	var ingressNames []string

	for _, ingress := range ingresses {
		ingressNames = append(ingressNames, ingress.ObjectMeta.Name)
	}

	return ingressNames
}
