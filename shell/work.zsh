work() {
  if [[ "$1" == "cd" ]]; then
    local dir
    dir=$(command work cd "${@:2}") && cd "$dir"
  else
    command work "$@"
  fi
}

# Register completions through the shell function.
eval "$(command work completion zsh)"
compdef _work work
