// Copyright 2020 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/okteto/okteto/pkg/analytics"
	"github.com/okteto/okteto/pkg/cmd/build"
	"github.com/okteto/okteto/pkg/cmd/down"
	"github.com/okteto/okteto/pkg/errors"
	k8Client "github.com/okteto/okteto/pkg/k8s/client"
	"github.com/okteto/okteto/pkg/k8s/deployments"
	"github.com/okteto/okteto/pkg/k8s/namespaces"
	"github.com/okteto/okteto/pkg/k8s/services"
	"github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/model"
	"github.com/okteto/okteto/pkg/okteto"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
)

//Push builds, pushes and redeploys the target deployment
func Push() *cobra.Command {
	var devPath string
	var namespace string
	var imageTag string
	var autoDeploy bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Builds, pushes and redeploys source code to the target deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("starting push command")

			if k8Client.InCluster() {
				return errors.ErrNotInCluster
			}

			dev, err := loadDev(devPath)
			if err != nil {
				return err
			}

			if err := dev.UpdateNamespace(namespace); err != nil {
				return err
			}
			c, _, configNamespace, err := k8Client.GetLocal()
			if err != nil {
				return err
			}
			if dev.Namespace == "" {
				dev.Namespace = configNamespace
			}
			oktetoRegistryURL := ""
			n, err := namespaces.Get(dev.Namespace, c)
			if err == nil {
				if namespaces.IsOktetoNamespace(n) {
					oktetoRegistryURL, err = okteto.GetRegistry()
					if err != nil {
						return err
					}
				}
			}

			if err := runPush(dev, autoDeploy, imageTag, oktetoRegistryURL, c); err != nil {
				analytics.TrackPush(false, oktetoRegistryURL)
				return err
			}

			log.Success("Source code pushed to the development environment '%s'", dev.Name)
			log.Println()

			analytics.TrackPush(true, oktetoRegistryURL)
			log.Info("completed push command")
			return nil
		},
	}

	cmd.Flags().StringVarP(&devPath, "file", "f", defaultManifest, "path to the manifest file")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace where the push command is executed")
	cmd.Flags().StringVarP(&imageTag, "tag", "t", "", "image tag to build, push and redeploy")
	cmd.Flags().BoolVarP(&autoDeploy, "deploy", "d", false, "create deployment when it doesn't exist in a namespace")
	return cmd
}

func runPush(dev *model.Dev, autoDeploy bool, imageTag, oktetoRegistryURL string, c *kubernetes.Clientset) error {
	create := false
	d, err := deployments.Get(dev, dev.Namespace, c)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if !autoDeploy {
			if err := askIfDeploy(dev.Name, dev.Namespace); err != nil {
				return err
			}
		}
		d = dev.GevSandbox()
		create = true
	}

	buildKitHost, isOktetoCluster, err := build.GetBuildKitHost()
	if err != nil {
		return err
	}
	if create {
		if imageTag == "" && !isOktetoCluster {
			return fmt.Errorf("you need to specify the image tag to build with the '-t' argument")
		}
	}

	imageTag = build.GetImageTag(dev, imageTag, d, oktetoRegistryURL)
	log.Infof("pushing with image tag %s", imageTag)

	var imageDigest string
	imageDigest, err = build.Run(buildKitHost, isOktetoCluster, ".", "Dockerfile", imageTag, "", false, nil)
	if err != nil {
		return fmt.Errorf("error building image '%s': %s", imageTag, err)
	}
	if imageDigest != "" {
		imageWithoutTag := build.GetRepoNameWithoutTag(imageTag)
		imageTag = fmt.Sprintf("%s@%s", imageWithoutTag, imageDigest)
	}

	spinner := newSpinner(fmt.Sprintf("Pushing source code to the development environment '%s'...", dev.Name))
	spinner.start()
	defer spinner.stop()
	if create {
		if err := createServiceAndDeployment(dev, d, imageTag, c); err != nil {
			return err
		}
	} else {
		err = down.Run(dev, imageTag, d, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func createServiceAndDeployment(dev *model.Dev, d *appsv1.Deployment, imageTag string, c *kubernetes.Clientset) error {
	d.Spec.Template.Spec.Containers[0].Image = imageTag
	if err := deployments.Deploy(d, true, c); err != nil {
		return err
	}
	if err := services.CreateDev(dev, c); err != nil {
		return err
	}
	return nil
}
