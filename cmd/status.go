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
	"github.com/okteto/okteto/pkg/cmd/status"
	"github.com/okteto/okteto/pkg/errors"
	k8Client "github.com/okteto/okteto/pkg/k8s/client"
	"github.com/okteto/okteto/pkg/log"
	"github.com/spf13/cobra"
)

//Status returns the status of the synchronization process
func Status() *cobra.Command {
	var devPath string
	var namespace string
	var showInfo bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: fmt.Sprintf("Status of the synchronization process"),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("starting status command")
			analytics.TrackStatus(true, showInfo)

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

			_, _, namespace, err = k8Client.GetLocal()
			if err != nil {
				return err
			}

			if dev.Namespace == "" {
				dev.Namespace = namespace
			}

			err = status.Run(dev, showInfo)
			analytics.TrackStatus(err == nil, showInfo)
			return err
		},
	}
	cmd.Flags().StringVarP(&devPath, "file", "f", defaultManifest, "path to the manifest file")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace where the up command is executing")
	cmd.Flags().BoolVarP(&showInfo, "info", "i", false, "show syncthing links for troubleshooting the synchronization service")
	return cmd
}
