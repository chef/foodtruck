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
package foodtruck

import (
	"fmt"
	"os"
	"time"

	"github.com/chef/foodtruck/connectors/azeventhub"
)

type Connector interface {
	Register(string) error
	StartListener() (chan string, error)
	StopListener() error
	Deregister() error
	SendOrder(order string) error
}

type Provider interface {
}

type Order struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Policy   string `json:"policy"`
	Change   Change `json:"change"`
}

type ChefPolicy struct {
	PolicyFileArchive string
	InspecArchive     string
	ParameterFile     string
}

type Change struct {
	Ticket      string    `json:"ticket"`
	WindowStart time.Time `json:"start"`
	WindowStop  time.Time `json:"stop"`
}

var c Connector

func Register() {
	queue, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	c = azeventhub.NewAzureEventHub(int64(2), int64(1))
	err = c.Register(queue)
	if err != nil {
		panic(err)
	}
}

func Listen() {
	orders, err := c.StartListener()
	if err != nil {
		panic(err)
	}
	o := <-orders

	fmt.Println(o)
}

func Send() {
	err := c.SendOrder("This is a test")
	if err != nil {
		panic(err)
	}
}
