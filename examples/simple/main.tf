#
# Private Bucket
#

module "s3_private_bucket" {
  source = "../../"

  bucket                   = var.test_name
  use_account_alias_prefix = false
  logging_bucket           = module.s3_logs.aws_logs_bucket
}

#
# Logging Bucket
#

module "s3_logs" {
  source  = "trussworks/logs/aws"
  version = "~> 4"

  s3_bucket_name = var.logging_bucket
  region         = var.region

  default_allow = false
}
