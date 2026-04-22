data "coder_secret" "my_token" {
  env          = "MY_TOKEN"
  help_message = "Personal access token injected as the environment variable MY_TOKEN"
}

data "coder_secret" "my_cert" {
  file         = "~/my-cert.pem"
  help_message = "Certificate chain injected as the file ~/my-cert.pem"
}

# Use the secret value in an agent startup script.
resource "coder_script" "setup" {
  agent_id = coder_agent.main.id
  script   = "echo ${data.coder_secret.my_token.value}"
}
