data "coder_workspace" "me" {
}

resource "kubernetes_pod" "dev" {
  count = data.coder_workspace.me.start_count
}

resource "tls_private_key" "example_key_pair" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "coder_metadata" "pod_info" {
  count       = data.coder_workspace.me.start_count
  resource_id = kubernetes_pod.dev[0].id
  # (Enterprise-only) this resource consumes 200 quota units
  daily_cost = 200
  item {
    key   = "description"
    value = "This description will show up in the Coder dashboard."
  }
  item {
    key   = "pod_uid"
    value = kubernetes_pod.dev[0].uid
  }
  item {
    key   = "public_key"
    value = tls_private_key.example_key_pair.public_key_openssh
    # The value of this item will be hidden from view by default
    sensitive = true
  }
}
