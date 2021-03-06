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

package ssh

import (
	"golang.org/x/crypto/ssh"
)

func getSSHClientConfig(user string) *ssh.ClientConfig {
	sshConfig := &ssh.ClientConfig{
		User: user,
		// skipcq GSC-G106
		// Ignoring this issue since the remote server doesn't have a set identity, and it's already secured by the
		// port-forward tunnel to the kubernetes cluster.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return sshConfig
}
