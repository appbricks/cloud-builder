#
# @recipe_description: Simple Test Recipe for Google
#

variable "test_simple_input_1" {
  type        = "string"
  description = "Simple test content to write to file 1"
}

variable "test_simple_input_2" {
  type        = "string"
  description = "Simple test content to write to file 2"
}

resource "local_file" "basic-test" {
  content  = "test data : ${var.test_simple_input_1} &  ${var.test_simple_input_2}"
  filename = "./test-simple.data"
}
