#
# @recipe_description: Basic Test Recipe for AWS
#

# Cloud Builder bastion recipe identifier
#
# @is_bastion: true

# Resource identifiers
#
# @resource_instance_list: instance1,instance2,instance3
# @resource_instance_data_list: data1,data2

# @display_name: Test Input #1
# @accepted_values: aa,bb,cc,dd
# @accepted_values_message: Error value #1
# @value_inclusion_filter: 
# @value_inclusion_filter_message:
# @value_exclusion_filter:
# @value_exclusion_filter_message:
# @environment_variables:
# @depends_on:
# @sensitive:
# @target_key: true
# @order: 1
#
variable "test_input_1" {
  type        = string
  description = "Description for Test Input #1"
}

# @display_name: Test Input #2
# @value_inclusion_filter: ^(appbricks)?cookbook
# @value_inclusion_filter_message: Error value inclusion #2
# @value_exclusion_filter: appbricks$
# @value_exclusion_filter_message: Error value exclusion #2
# @target_key: true
# @order: 3
#
variable "test_input_2" {
  type        = string
  description = "Description for Test Input #2"
}

# @display_name: Test Input #3
# @sensitive: true
# @order: 2
#
variable "test_input_3" {
  type        = string
  default     = "abcd3"
  description = "Description for Test Input #3"
}

# @display_name: Test Input #5
# @sensitive: true
# @order: 0
#
variable "test_input_5" {
  type        = string
  description = "Description for Test Input #5"
}

# @display_name: Test Input #7
# @accepted_values: +iaas_regions
# @accepted_values_message: Error! not a valid region
# @order: 4
#
variable "test_input_7" {
  type        = string
  default     = "us-east-1"
  description = "Description for Test Input #7"
}

module "label" {
  source     = "git::https://github.com/cloudposse/terraform-null-label.git?ref=master"

  namespace   = "abc"
  stage       = "xyz"
  name        = "cloudbuilder"
  delimiter   = "-"
}

resource "local_file" "basic-test" {
  content  = "test data : ${var.test_input_1}"
  filename = "./test.data"
}

output "test_output_1" {
  value = "${local_file.basic-test.content} 1"
}

output "test_output_2" {
  value = "${local_file.basic-test.content} 2"
}
