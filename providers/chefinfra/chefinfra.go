/*
Copyright © 2020 Chef Software, Inc <success@chef.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chefinfra

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// func executePolicyArchive() {}
// func passInAttrAsJson() {}

type chefInfraDefinition struct {
	args           []string
	policyURL      string `json:"policy_url"`
	policySum      string `json:"policy_sum"`
	jsonAttributes string `json:"attributes"`
}

// New creates a new Chef Infra Definition that can be used to execute Chef Infra Client.
func New(m map[string]interface{}) (*chefInfraDefinition, error) {
	chef := chefInfraDefinition(m)
	path, err := exec.LookPath("chef-client")
	if err != nil {
		return nil, err
	}

	append(chef.args, path)
	return chef, nil
}

// Prepare prepares the chef-infra-client commands and downloads and processes the archives.
func (chef *chefInfraDefinition) Prepare() error {
	if chef.policyURL != nil {
		append(chef.args, "--local-mode")

		s := strings.Split(chef.policyURL, "/")
		filepath := ".foodtruck/chefinfra/" + s[len(s)-1]
		append(chef.args, "--recipe-url", filepath)

		err := downloadFile(chef.policyURL, filepath)
		if err != nil {
			return err
		}
	}
	if chef.jsonAttributes != nil {
		append(chef.args, "--json-attributes", ".foodtruck/chefinfra/dna.json")

		o, err := json.Marshal(chef.jsonAttributes)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(".foodtruck/chefinfra/dna.json", o, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}

func downloadFile(path, filepath, checksum string) error {
	resp, err := http.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	if checksum != nil {
		if sha256.Sum256(out.Read()) != checksum {
			return errors.New("checksum of downlaoded file does not match provided sha256 checksum")
		}
	}
	return nil
}

// Execute executes the Chef Infra Client with the provided parameters
func (chef *chefInfraDefinition) Execute() error {
	cmd := exec.Command(chef.path, chef.args)
	cmd.Dir = ".foodtruck/chefinfra"
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
