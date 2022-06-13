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

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCloudStackInstanceResetPassword() *schema.Resource {
	return &schema.Resource{
		Update: resourceCloudStackInstancePasswordUpdate,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackInstancePasswordUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	virtualMachineId := d.Get("id")

	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualMachineId.(string))
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Virtual Machine with id: %s no longer exists", virtualMachineId.(string))
			d.SetId("")
			return nil
		}
		return err
	}

	p := cs.VirtualMachine.NewResetPasswordForVirtualMachineParams(virtualMachineId.(string))
	_, err = cs.VirtualMachine.ResetPasswordForVirtualMachine(p)

	if err != nil {
		return fmt.Errorf("Failed to reset password for virtual machine %s [%s] due to : %v", vm.Name, vm.Id, err)
	}

	d.SetPartial("password")
	return resourceCloudStackInstanceRead(d, meta)
}
