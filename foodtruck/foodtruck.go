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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/chef/foodtruck/connectors/azeventhub"
	"github.com/chef/foodtruck/providers/chefinfra"
	"github.com/chef/foodtruck/providers/chefinspec"
	"github.com/chef/foodtruck/providers/mock"
	"github.com/google/uuid"
)

type Connector interface {
	Register(string) error
	StartListener() (chan []byte, error)
	StopListener() error
	Deregister() error
	SendOrder(order []byte) error
}

type Provider interface {
	New(map[string]interface{}) (*Provider, error)
	Prepare() error
	Execute() error
	Clean() error
}

type Order struct {
	ID       string   `json:"id"`
	Policies []Policy `json:"policies"`
	Change   Change   `json:"change"`
}

type Policy struct {
	Provider   string                 `json:"provider"`
	Definition map[string]interface{} `json:"definition"`
}

type Change struct {
	Ticket      string    `json:"ticket"`
	WindowStart time.Time `json:"start"`
	WindowStop  time.Time `json:"stop"`
}

var c Connector

func Init() {
	ensureDir(".foodtruck")
	ensureDir(".foodtruck/orders")

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

func ensureDir(path string) {
	dir, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.Mkdir(path, 0700)
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	} else if !dir.IsDir() {
		panic("Path " + path + " exists and is not a directory!")
	}
}

func Listen() {
	orders, err := c.StartListener()
	if err != nil {
		panic(err)
	}
	for {
		o := <-orders
		receive(o)
	}
}

func receive(o []byte) {
	order := Order{}
	json.Unmarshal(o, &order)
	fmt.Printf("Order %v Received!\n", order.ID)
	err := ioutil.WriteFile(".foodtruck/orders/"+order.ID+".json", o, 0700)
	if err != nil {
		panic(err)
	}
	processOrder(order)
}

func Send() {
	order := `{"id":"` + uuid.New().String() + `","policies":[{"provider":"mock","definition":{"attrib1":"abc","attrib2":"123","nested":{"attrib3":"a1"}}}],"change":{"ticket":"abc123","start":"2020-01-01 00:00:00", "end":"2021-01-01 00:00:00"}}`
	err := c.SendOrder([]byte(order))
	if err != nil {
		panic(err)
	}
}

func processOrder(o Order) {
	var p Provider

	for _, policy := range o.Policies {
		switch policy.Provider {
		case "chefinfra":
			ensureDir(".foodtruck/chefinfra")
			p = chefinfra.New(policy.Definition)
		case "chefinspec":
			ensureDir(".foodtruck/chefinspec")
			p = chefinspec.New(policy.Definition)
		case "mock":
			p = mock.New()
		}
		err := p.Execute()
		if err != nil {
			panic(err)
		}
	}
}
