package deployments

import (
	"encoding/json"
	"fmt"

	"github.com/okteto/okteto/pkg/errors"
	"github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Get returns a deployment object given its name and namespace
func Get(dev *model.Dev, namespace string, c *kubernetes.Clientset) (*appsv1.Deployment, error) {
	if namespace == "" {
		return nil, fmt.Errorf("empty namespace")
	}

	var d *appsv1.Deployment
	var err error

	if len(dev.Labels) == 0 {
		d, err = c.AppsV1().Deployments(namespace).Get(dev.Name, metav1.GetOptions{})
		if err != nil {
			log.Debugf("error while retrieving deployment %s/%s: %s", namespace, dev.Name, err)
			return nil, err
		}
	} else {
		deploys, err := c.AppsV1().Deployments(namespace).List(
			metav1.ListOptions{
				LabelSelector: dev.LabelsSelector(),
			},
		)
		if err != nil {
			return nil, err
		}
		if len(deploys.Items) == 0 {
			return nil, fmt.Errorf("Deployment not found")
		}
		if len(deploys.Items) > 1 {
			return nil, fmt.Errorf("Found '%d' deployments instead of 1", len(deploys.Items))
		}
		d = &deploys.Items[0]
	}

	return d, nil
}

//GetAll fills all the deployments pointed by a dev environment
func GetAll(dev *model.Dev, c *kubernetes.Clientset) error {
	d, err := Get(dev, dev.Namespace, c)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		dev.Deployment = d
	}

	for _, s := range dev.Services {
		d, err := Get(s, dev.Namespace, c)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		s.Deployment = d
	}
	return nil
}

//Deploy creates or updates a deployment
func Deploy(d *appsv1.Deployment, forceCreate bool, client *kubernetes.Clientset) error {
	if forceCreate {
		if err := create(d, client); err != nil {
			return err
		}
	} else {
		if err := update(d, client); err != nil {
			return err
		}
	}
	return nil
}

//TraslateDevMode translates the deployment manifests to put them in dev mode
func TraslateDevMode(dev *model.Dev, nodeName string) error {
	err := translate(dev, dev, nodeName)
	if err != nil {
		return err
	}

	for i, s := range dev.Services {
		if s.Deployment == nil {
			return fmt.Errorf("Deployment for service number %d not found", i)
		}
		err := translate(dev, s, nodeName)
		if err != nil {
			return err
		}
	}
	return nil
}

//IsDevModeOn returns if a deployment is in devmode
func IsDevModeOn(d *appsv1.Deployment) bool {
	annotations := d.GetObjectMeta().GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[oktetoDevAnnotation]
	return ok
}

//IsAutoCreate returns if the deplloyment is created from scratch
func IsAutoCreate(d *appsv1.Deployment) bool {
	annotations := d.GetObjectMeta().GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[oktetoAutoCreateAnnotation]
	return ok
}

// DevModeOff deactivates dev mode for d
func DevModeOff(d *appsv1.Deployment, c *kubernetes.Clientset) error {
	dManifest := getAnnotation(d.GetObjectMeta(), oktetoDeploymentAnnotation)
	if len(dManifest) == 0 {
		log.Infof("%s/%s is not an okteto environment", d.Namespace, d.Name)
		return nil
	}

	dOrig := &appsv1.Deployment{}
	if err := json.Unmarshal([]byte(dManifest), dOrig); err != nil {
		return fmt.Errorf("malformed manifest: %s", err)
	}

	dOrig.ResourceVersion = ""
	if err := update(dOrig, c); err != nil {
		return err
	}

	return nil
}

func create(d *appsv1.Deployment, c *kubernetes.Clientset) error {
	log.Debugf("creating deployment %s/%s", d.Namespace, d.Name)
	_, err := c.AppsV1().Deployments(d.Namespace).Create(d)
	if err != nil {
		return err
	}
	return nil
}

func update(d *appsv1.Deployment, c *kubernetes.Clientset) error {
	log.Debugf("updating deployment %s/%s", d.Namespace, d.Name)
	_, err := c.AppsV1().Deployments(d.Namespace).Update(d)
	if err != nil {
		return err
	}
	return nil
}