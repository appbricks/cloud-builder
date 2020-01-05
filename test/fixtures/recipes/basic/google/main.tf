#
# @recipe_description: Basic Test Recipe for Google
#

variable "test_input" {
  type        = "string"
  description = "Basic test content to write to file"
}

resource "local_file" "basic-test" {
  content  = "test data : ${var.test_input}"
  filename = "./test.data"
}

output "test_output_1" {
  value = "${local_file.basic-test.content} 1"
}

output "test_output_2" {
  value = "${local_file.basic-test.content} 2"
}

# Google provider

data "google_client_config" "current" {}

output "project" {
  value = "${data.google_client_config.current.project}"
}
