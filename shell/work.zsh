work() {
  if [[ "$1" == "cd" ]]; then
    local dir
    dir=$(command work cd "${@:2}") && cd "$dir"
  else
    command work "$@"
  fi
}

# Register completions.
eval "$(command work completion zsh)"
compdef _work work
eval "$(command task completion zsh)"
compdef _task task
eval "$(command agent completion zsh)"
compdef _agent agent
