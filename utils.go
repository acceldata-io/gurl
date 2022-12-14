// Acceldata Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 	Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func isInSlice[T comparable](a T, list []T) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func removeFromSlice[T comparable](s T, l []T) []T {
	result := []T{}
	for _, e := range l {
		if e != s {
			result = append(result, e)
		}
	}

	return result
}

func isKRBDepsAvail() error {
	//
	deps := []string{"kinit", "klist", "awk", "grep", "head"}
	krbConfigFromEnv := strings.TrimSpace(os.Getenv("KRB5_CONFIG"))

	if _, err := os.Stat(defaultShell); err != nil {
		return fmt.Errorf("cannot find or access '" + defaultShell + "' because " + err.Error())
	}

	if krbConfigFromEnv == "" {
		if _, err := os.Stat(defaultKRBConfig); err != nil {
			if _, err := os.Stat(secondaryKRBConfig); err != nil {
				return fmt.Errorf("cannot find or access '" + defaultKRBConfig + "' OR '" + secondaryKRBConfig + "' and the ENV variable '" + krbConfigFromEnv + "' is empty")
			}
		}
	} else {
		if _, err := os.Stat(krbConfigFromEnv); err != nil {
			return fmt.Errorf("got custom KRB config path from the ENV variable '" + krbConfigFromEnv + "' and is not accessible because " + err.Error())
		}
	}

	//
	for _, dep := range deps {
		//
		var outb, errb bytes.Buffer
		tryKlist := exec.Command(defaultShell, "-c", "which "+dep)
		tryKlist.Stdout = &outb
		tryKlist.Stderr = &errb
		if err := tryKlist.Run(); err != nil {
			return fmt.Errorf("cannot check for the dependency binary '" + dep + "' using the command '" + fmt.Sprintf(defaultShell+" -c"+" which "+dep) + "' because " + errb.String() + err.Error())
		}

		//
		if strings.TrimSpace(outb.String()) == "" {
			return fmt.Errorf("cannot find the binary '" + dep + "' in the OS path")
		}
	}
	return nil
}
