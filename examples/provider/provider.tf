terraform {
  required_providers {
    krutrim = {
      source  = "ola-silicon/krutrim"
      version = "1.0.0"
    }
  }
}

provider "krutrim" {
  base_url = "https://cloud.olakrutrim.com"

  email    = "enter_email_here"
  password = "enter the password"

  is_root_user = true
}