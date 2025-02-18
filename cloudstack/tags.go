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
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/apache/cloudstack-go/v2/cloudstack"
)

// tagsSchema returns the schema to use for tags
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Computed: true,
	}
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(cs *cloudstack.CloudStackClient, d *schema.ResourceData, resourcetype string) error {
	if tags, ok := d.GetOk("tags"); ok {
		p := cs.Resourcetags.NewCreateTagsParams(
			[]string{d.Id()},
			resourcetype, tagsFromSchema(tags.(map[string]interface{})),
		)
		_, err := cs.Resourcetags.CreateTags(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateTags is a helper to update only when tags field change tags
// field to be named "tags"
func updateTags(cs *cloudstack.CloudStackClient, d *schema.ResourceData, resourcetype string) error {
	oraw, nraw := d.GetChange("tags")
	o := oraw.(map[string]interface{})
	n := nraw.(map[string]interface{})

	remove, create := diffTags(tagsFromSchema(o), tagsFromSchema(n))
	log.Printf("[DEBUG] tags to remove: %v", remove)
	log.Printf("[DEBUG] tags to create: %v", create)

	// First remove any obsolete tags
	if len(remove) > 0 {
		log.Printf("[DEBUG] Removing tags: %v from %s", remove, d.Id())
		p := cs.Resourcetags.NewDeleteTagsParams([]string{d.Id()}, resourcetype)
		p.SetTags(remove)
		_, err := cs.Resourcetags.DeleteTags(p)
		if err != nil {
			return err
		}
	}

	// Then add any new tags
	if len(create) > 0 {
		log.Printf("[DEBUG] Creating tags: %v for %s", create, d.Id())
		p := cs.Resourcetags.NewCreateTagsParams([]string{d.Id()}, resourcetype, create)
		_, err := cs.Resourcetags.CreateTags(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// diffTags takes the old and the new tag sets and returns the difference of
// both. The remaining tags are those that need to be removed and created
func diffTags(oldTags, newTags map[string]string) (map[string]string, map[string]string) {
	for k, old := range oldTags {
		new, ok := newTags[k]
		if ok && old == new {
			// We should avoid removing or creating tags we already have
			delete(oldTags, k)
			delete(newTags, k)
		}
	}

	return oldTags, newTags
}

// tagsFromSchema takes the raw schema tags and returns them as a
// properly asserted map[string]string
func tagsFromSchema(m map[string]interface{}) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}
