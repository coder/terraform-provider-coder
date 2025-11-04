provider "coder" {}

data "coder_workspace" "me" {}
data "coder_task" "me" {}

resource "coder_ai_task" "task" {
  count  = data.coder_task.me.enabled ? data.coder_workspace.me.start_count : 0
  app_id = module.example-agent.task_app_id
}

module "example-agent" {
  count  = data.coder_task.me.enabled ? data.coder_workspace.me.start_count : 0
  prompt = data.coder_ai_task.me.prompt
}
