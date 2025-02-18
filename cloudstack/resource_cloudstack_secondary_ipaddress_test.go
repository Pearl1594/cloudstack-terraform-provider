//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/apache/cloudstack-go/v2/cloudstack"
)

func TestAccCloudStackSecondaryIPAddress_basic(t *testing.T) {
	var ip cloudstack.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudStackSecondaryIPAddress_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSecondaryIPAddressExists(
						"cloudstack_secondary_ipaddress.foo", &ip),
				),
			},
		},
	})
}

func TestAccCloudStackSecondaryIPAddress_fixedIP(t *testing.T) {
	var ip cloudstack.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudStackSecondaryIPAddress_fixedIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSecondaryIPAddressExists(
						"cloudstack_secondary_ipaddress.foo", &ip),
					testAccCheckCloudStackSecondaryIPAddressAttributes(&ip),
					resource.TestCheckResourceAttr(
						"cloudstack_secondary_ipaddress.foo", "ip_address", "10.1.1.123"),
				),
			},
		},
	})
}

func testAccCheckCloudStackSecondaryIPAddressExists(
	n string, ip *cloudstack.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

		virtualmachine, ok := rs.Primary.Attributes["virtual_machine_id"]
		if !ok {
			virtualmachine, ok = rs.Primary.Attributes["virtual_machine"]
		}

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine)
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return fmt.Errorf("Instance not found")
			}
			return err
		}

		nicid, ok := rs.Primary.Attributes["nic_id"]
		if !ok {
			nicid, ok = rs.Primary.Attributes["nicid"]
		}
		if !ok {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				ip.Ipaddress = sip.Ipaddress
				ip.Nicid = l.Nics[0].Id
				return nil
			}
		}

		return fmt.Errorf("IP address not found")
	}
}

func testAccCheckCloudStackSecondaryIPAddressAttributes(
	ip *cloudstack.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if ip.Ipaddress != "10.1.1.123" {
			return fmt.Errorf("Bad IP address: %s", ip.Ipaddress)
		}
		return nil
	}
}

func testAccCheckCloudStackSecondaryIPAddressDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_secondary_ipaddress" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		virtualmachine, ok := rs.Primary.Attributes["virtual_machine_id"]
		if !ok {
			virtualmachine, ok = rs.Primary.Attributes["virtual_machine"]
		}

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine)
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return nil
			}
			return err
		}

		nicid, ok := rs.Primary.Attributes["nic_id"]
		if !ok {
			nicid, ok = rs.Primary.Attributes["nicid"]
		}
		if !ok {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				return fmt.Errorf("IP address %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}

	return nil
}

const testAccCloudStackSecondaryIPAddress_basic = `
resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "10.1.1.0/24"
  network_offering = "DefaultIsolatedNetworkOfferingWithSourceNatService"
  zone = "Sandbox-simulator"
}

resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "Medium Instance"
  network_id = "${cloudstack_network.foo.id}"
  template = "CentOS 5.6 (64-bit) no GUI (Simulator)"
  zone = "Sandbox-simulator"
  expunge = true
}

resource "cloudstack_secondary_ipaddress" "foo" {
	virtual_machine_id = "${cloudstack_instance.foobar.id}"
} `

const testAccCloudStackSecondaryIPAddress_fixedIP = `
resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "10.1.1.0/24"
  network_offering = "DefaultIsolatedNetworkOfferingWithSourceNatService"
  zone = "Sandbox-simulator"
}

resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "Medium Instance"
  network_id = "${cloudstack_network.foo.id}"
  template = "CentOS 5.6 (64-bit) no GUI (Simulator)"
  zone = "Sandbox-simulator"
  expunge = true
}

resource "cloudstack_secondary_ipaddress" "foo" {
	ip_address = "10.1.1.123"
	virtual_machine_id = "${cloudstack_instance.foobar.id}"
}`
