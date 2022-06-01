#
# @recipe_description: Simple Test Recipe for Google
#

# @display_name: Deployment Name
# @accepted_values: 
# @accepted_values_message: Name Value Error
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
variable "name" {
  type        = string
  description = "Description for deployment Name Input"
}

# @display_name: Test Simple Input #1
# @accepted_values: 
# @accepted_values_message: Error value #1
# @value_inclusion_filter: 
# @value_inclusion_filter_message:
# @value_exclusion_filter:
# @value_exclusion_filter_message:
# @environment_variables:
# @depends_on:
# @sensitive:
# @target_key: true
# @order: 2
#
variable "test_simple_input_1" {
  type        = string
  description = "Simple test content to write to file 1"
}

# @display_name: Test Simple Input #2
# @accepted_values: 
# @accepted_values_message: Error value #2
# @value_inclusion_filter: 
# @value_inclusion_filter_message:
# @value_exclusion_filter:
# @value_exclusion_filter_message:
# @environment_variables:
# @depends_on:
# @sensitive:
# @target_key:
# @order: 3
#
variable "test_simple_input_2" {
  type        = string
  description = "Simple test content to write to file 2"
}

resource "local_file" "basic-test" {
  content  = "test data : ${var.test_simple_input_1} &  ${var.test_simple_input_2}"
  filename = "./test-simple.data"
}
