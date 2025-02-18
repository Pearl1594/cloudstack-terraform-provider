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
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/apache/cloudstack-go/v2/cloudstack"
)

func resourceCloudStackSecondaryIPAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackSecondaryIPAddressCreate,
		Read:   resourceCloudStackSecondaryIPAddressRead,
		Delete: resourceCloudStackSecondaryIPAddressDelete,

		Schema: map[string]*schema.Schema{
			"ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"nic_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackSecondaryIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	nicid, ok := d.GetOk("nic_id")
	if !ok {
		virtualmachineid := d.Get("virtual_machine_id").(string)

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				log.Printf("[DEBUG] Virtual Machine %s does no longer exist", virtualmachineid)
				d.SetId("")
				return nil
			}
			return err
		}

		nicid = vm.Nic[0].Id
	}

	// Create a new parameter struct
	p := cs.Nic.NewAddIpToNicParams(nicid.(string))

	// If there is a ipaddres supplied, add it to the parameter struct
	if ipaddress, ok := d.GetOk("ip_address"); ok {
		p.SetIpaddress(ipaddress.(string))
	}

	ip, err := cs.Nic.AddIpToNic(p)
	if err != nil {
		return err
	}

	d.SetId(ip.Id)

	return resourceCloudStackSecondaryIPAddressRead(d, meta)
}

func resourceCloudStackSecondaryIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	virtualmachineid := d.Get("virtual_machine_id").(string)

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Virtual Machine %s does no longer exist", virtualmachineid)
			d.SetId("")
			return nil
		}
		return err
	}

	nicid, ok := d.GetOk("nic_id")
	if !ok {
		nicid = vm.Nic[0].Id
	}

	p := cs.Nic.NewListNicsParams(virtualmachineid)
	p.SetNicid(nicid.(string))

	l, err := cs.Nic.ListNics(p)
	if err != nil {
		return err
	}

	if l.Count == 0 {
		log.Printf("[DEBUG] NIC %s does no longer exist", d.Get("nic_id").(string))
		d.SetId("")
		return nil
	}

	if l.Count > 1 {
		return fmt.Errorf("Found more then one possible result: %v", l.Nics)
	}

	for _, ip := range l.Nics[0].Secondaryip {
		if ip.Id == d.Id() {
			d.Set("ip_address", ip.Ipaddress)
			d.Set("nic_id", l.Nics[0].Id)
			d.Set("virtual_machine_id", l.Nics[0].Virtualmachineid)
			return nil
		}
	}

	log.Printf("[DEBUG] IP %s no longer exist", d.Get("ip_address").(string))
	d.SetId("")

	return nil
}

func resourceCloudStackSecondaryIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.Nic.NewRemoveIpFromNicParams(d.Id())

	log.Printf("[INFO] Removing secondary IP address: %s", d.Get("ip_address").(string))
	if _, err := cs.Nic.RemoveIpFromNic(p); err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error removing secondary IP address: %s", err)
	}

	return nil
}
