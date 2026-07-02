// Copyright 2022 Linka Cloud  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package d2vm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerfileKernelFlag verifies that the --kernel flag is honored by the
// Debian template (shared with Kali): the kernel package is only installed
// when Kernel=true, while apt-get update and the /boot symlink cleanup must
// remain so that later RUN layers (e.g. systemd install) still work.
func TestDockerfileKernelFlag(t *testing.T) {
	// Both Debian and Kali render through debianDockerfileTemplate.
	releases := map[string]OSRelease{
		"debian": {ID: ReleaseDebian, Name: "Debian GNU/Linux", VersionID: "12", Version: "12 (bookworm)", VersionCodeName: "bookworm"},
		"kali":   {ID: ReleaseKali, Name: "Kali GNU/Linux Rolling", VersionID: "2023.3", Version: "2023.3"},
	}
	for name, rel := range releases {
		rel := rel
		for _, kernel := range []bool{true, false} {
			kernel := kernel
			t.Run(name+"_kernel_"+boolStr(kernel), func(t *testing.T) {
				d, err := NewDockerfile(rel, "image:latest", "", NetworkManagerIfupdown2, false, false, false, kernel)
				require.NoError(t, err)
				var buf bytes.Buffer
				require.NoError(t, d.Render(&buf))
				out := buf.String()
				if kernel {
					assert.Contains(t, out, "linux-image-amd64", "kernel should be installed when Kernel=true")
				} else {
					assert.NotContains(t, out, "linux-image-amd64", "kernel should not be installed when Kernel=false")
					// apt-get update must remain so the subsequent systemd install has a populated cache.
					assert.Contains(t, out, "apt-get update", "apt-get update must remain for later RUN layers")
					assert.Contains(t, out, "find /boot -type l", "boot symlink cleanup must remain")
				}
			})
		}
	}
}

// TestDockerfileKernelFlagCentOS verifies that the CentOS template honors the
// --kernel flag: the `kernel` package is only installed when Kernel=true, and
// the boot-file selection step uses the deterministic `ls -t | head -n1` form
// rather than the fragile `find` form that breaks when multiple kernels are
// installed.
func TestDockerfileKernelFlagCentOS(t *testing.T) {
	rel := OSRelease{ID: ReleaseCentOS, Name: "CentOS Stream", VersionID: "9", Version: "9"}
	for _, kernel := range []bool{true, false} {
		kernel := kernel
		t.Run("centos_kernel_"+boolStr(kernel), func(t *testing.T) {
			d, err := NewDockerfile(rel, "image:latest", "", NetworkManagerNone, false, false, false, kernel)
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, d.Render(&buf))
			out := buf.String()
			assert.Contains(t, out, "yum install", "base yum install must remain")
			assert.Contains(t, out, "ls -t /boot/vmlinuz-*", "boot selection must use deterministic ls -t form")
			assert.NotContains(t, out, "mv $(find", "boot selection must not use the fragile find form")
			if kernel {
				assert.Contains(t, out, "kernel \\", "kernel package should be installed when Kernel=true")
			} else {
				assert.NotContains(t, out, "kernel \\", "kernel package should not be installed when Kernel=false")
			}
		})
	}
}

// TestDockerfileKernelFlagAlpine verifies that the Alpine template honors the
// --kernel flag: the `linux-virt` package is only installed when Kernel=true.
func TestDockerfileKernelFlagAlpine(t *testing.T) {
	rel := OSRelease{ID: ReleaseAlpine, Name: "Alpine Linux", VersionID: "3.18", Version: "3.18.0"}
	for _, kernel := range []bool{true, false} {
		kernel := kernel
		t.Run("alpine_kernel_"+boolStr(kernel), func(t *testing.T) {
			d, err := NewDockerfile(rel, "image:latest", "", NetworkManagerIfupdown2, false, false, false, kernel)
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, d.Render(&buf))
			out := buf.String()
			if kernel {
				assert.Contains(t, out, "linux-virt", "linux-virt should be installed when Kernel=true")
			} else {
				assert.NotContains(t, out, "linux-virt", "linux-virt should not be installed when Kernel=false")
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
