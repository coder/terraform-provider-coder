provider "coder" {}

data "coder_script_order" "startup_dependencies" {
  rule {
    run   = ["coder_script.install_tools"]
    after = ["coder_script.clone_repo"]
  }

  rule {
    run      = ["coder_script.configure_workspace"]
    after    = ["module.bootstrap"]
    requires = "completion"
  }

  # module.cleanup has both start and stop scripts. phase keeps only the
  # stop scripts and suppresses the inferred-phase warning.
  rule {
    run   = ["coder_script.archive_logs"]
    after = ["module.cleanup"]
    phase = "stop"
  }
}
