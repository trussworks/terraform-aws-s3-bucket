variable "test_name" {
  type = string
}

variable "logging_bucket" {
  type = string
}

variable "enable_analytics" {
  type = bool
}

variable "cors_rules" {
  type    = list(any)
  default = []
}

variable "enable_versioning" {
  type = bool
}

variable "cors_rules" {
  type    = list(any)
  default = []
}
