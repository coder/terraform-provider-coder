data "coder_workspace" "me" {
}

resource "kubernetes_pod" "dev" {
  count = data.coder_workspace.me.start_count
}

resource "coder_metadata" "pod_info" {
  count = data.coder_workspace.me.start_count
  resource_id = kubernetes_pod.dev[0].id
  pair {
    key = "pod_uid"
    value = kubernetes_pod.dev[0].uid
  }
}
