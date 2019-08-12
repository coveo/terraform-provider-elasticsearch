package main

import (
	"context"
	"fmt"
	"testing"
	
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	
	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"
	elastic5 "gopkg.in/olivere/elastic.v5"
)

func TestAccElasticsearchXpackRole(t *testing.T) {
	randomName := "test" + acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	
	provider := Provider().(*schema.Provider)
	err := provider.Configure(&terraform.ResourceConfig{})
	if err != nil {
		t.Skipf("err: %s", err)
	}
	meta := provider.Meta()
	var allowed bool
	var implemented bool
	switch meta.(type) {
	case *elastic5.Client:
		allowed = false
	case *elastic7.Client:
		allowed = true
		implemented = false
	default:
		allowed = true
		implemented = true
	}
	
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) 
			if !allowed {
				t.Skip("Xpack only supported on ES >= 6")
			}
			if !implemented {
				t.Skip("XpackRoles not implemented for ES 7. Contributions welcomed")
			}
		},
		Providers:    testAccXPackProviders,
		CheckDestroy: testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckRoleExists("elasticsearch_xpack_role.test"),
					resource.TestCheckResourceAttr(
						"elasticsearch_xpack_role.test",
						"id",
						randomName,
					),
					resource.TestCheckResourceAttr(
						"elasticsearch_xpack_role.test",
						"cluster.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"elasticsearch_xpack_role.test",
						"metadata",
						//`{"foo":"bar"}`,
						"{}",
					),
				),
			},
			{
				Config: testAccRoleResource_Updated(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckRoleExists("elasticsearch_xpack_role.test"),
					resource.TestCheckResourceAttr(
						"elasticsearch_xpack_role.test",
						"metadata",
						`{"foo":"bar"}`,
					),
				),
			},
			{
				Config: testAccRoleResource_Global(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckRoleExists("elasticsearch_xpack_role.test"),
					resource.TestCheckResourceAttr(
						"elasticsearch_xpack_role.test",
						"global",
						`{"application":{"manage":{"applications":["testapp"]}}}`,
					),
				),
			},
		},
	})
}

func testAccCheckRoleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticsearch_xpack_role" {
			continue
		}
		
		meta := testAccXPackProvider.Meta()
		
		if client, ok := meta.(*elastic6.Client); ok {
			if _, err := client.XPackSecurityGetRole(rs.Primary.ID).Do(context.TODO()); err != nil {
				if elasticErr, ok := err.(*elastic6.Error); ok && elasticErr.Status == 404 {
					return nil
					} else {
						return fmt.Errorf("Role %q still exists", rs.Primary.ID)
					}
					} else {
						return err
					}
					
					} else {
						return fmt.Errorf("Unsupported client type : %v", meta)
					}
				}
				return nil
			}
			
			func testCheckRoleExists(name string) resource.TestCheckFunc {
				return func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("Not found: %s", name)
					}
					if rs.Primary.ID == "" {
						return fmt.Errorf("No role mapping ID is set")
					}
					
					meta := testAccXPackProvider.Meta()
					
					client := meta.(*elastic6.Client)
					_, err := client.XPackSecurityGetRole(rs.Primary.ID).Do(context.TODO())
					
					if err != nil {
						return err
					}
					
					return nil
				}
			}
			
			func testAccRoleResource(resourceName string) string {
				return fmt.Sprintf(` 
				resource "elasticsearch_xpack_role" "test" {
					role_name = "%s"
					indices {
						names 	   = ["testIndice"]
						privileges = ["testPrivilege"]
					}
					indices {
						names 	   = ["testIndice2"]
						privileges = ["testPrivilege2"]
					}
					cluster = [
					"all"
					]
					applications {
						application = "testapp"
						privileges = [ 
						"admin", 
						"read" 
						]
						resources = [ 
						"*" 
						]
					}
				}
				`, resourceName)
			}
			
			func testAccRoleResource_Updated(resourceName string) string {
				return fmt.Sprintf(`
				resource "elasticsearch_xpack_role" "test" {
					role_name = "%s"
					indices {
						names 	 = ["testIndice"]
						privileges = ["testPrivilege"]
					}
					indices {
						names 	 = ["testIndice2"]
						privileges = ["testPrivilege2"]
					}
					cluster = [
					"all"
					]
					applications {
						application = "testapp"
						privileges = [ 
						"admin", 
						"read",
						"delete", 
						]
						resources = [ 
						"*" 
						]
					}
					metadata = <<-EOF
					{
						"foo": "bar"
					}
					EOF
				}
				`, resourceName)
			}
			
			func testAccRoleResource_Global(resourceName string) string {
				return fmt.Sprintf(`
				resource "elasticsearch_xpack_role" "test" {
					role_name = "%s"
					indices {
						names 	 = ["testIndice"]
						privileges = ["testPrivilege"]
					}
					indices {
						names 	 = ["testIndice2"]
						privileges = ["testPrivilege2"]
					}
					cluster = [
					"all",
					]
					applications {
						application = "testapp"
						privileges = [ 
						"admin", 
						"read",
						"delete", 
						]
						resources = [ 
						"*" ,
						]
					}
					
					metadata = <<-EOF
					{
						"foo": "bar"
					}
					EOF
					
					
					global = <<-EOF
					{
						"application": {
							"manage": {    
								"applications": ["testapp"] 
							}
						}
					}
					EOF
				}
				`, resourceName)
			}
			