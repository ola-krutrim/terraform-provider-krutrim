terraform {
  required_providers {
    krutrim = {
      source  = "ola-krutrim/krutrim"
    }
  }
}

provider "krutrim" {
  base_url = "https://r1.staging.olakrutrim.com"

  email    = "sanchit.amar1@olakrutrim.com"
  password = "Krutrim@1234"

  is_root_user = true
}
