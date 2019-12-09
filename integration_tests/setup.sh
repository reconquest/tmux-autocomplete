#!/bin/bash

tests:clone "tmux-autocomplete" "bin/tests-tmux-autocomplete"
tests:clone "tests/bin/print-data" "bin/"
tests:clone "tests/bin/signal" "bin/"
tests:clone "tests/bash.rc" "."
tests:clone "tests/tmux.conf" "."

:tmux() {
    tests:debug tmux -L "tmux-autocomplete-tests" -f $(tests:get-tmp-dir)/tmux.conf "${@}"
    tmux -L "tmux-autocomplete-tests" -f $(tests:get-tmp-dir)/tmux.conf "${@}"
}

:tmux-kill() {
    :tmux kill-server || true
}

:tmux-start() {
    :tmux-kill
    :tmux start-server
}

:tmux-new() {
    local session=foo
    :tmux new-session -d -P -s "$session" \
        /bin/bash --rcfile $(tests:get-tmp-dir)/bash.rc
}

:tmux-type() {
    local session=foo
    :tmux send-keys -t "$session" "${@}"
}


:tmux-sh() {
    :tmux-type "${@}"$'\n'
    :tmux-type "signal"$'\n'
}

:tmux-cat() {
    local session=foo
    :tmux capture-pane "-pt" "$session" "${@:--e}"
}

:tmux-wait() {
    if [[ "${@}" ]]; then
        "${@}"
    fi

    :tmux wait-for signal
}

:tmux-complete() {
    coproc:run ta \
        :tmux run $(tests:get-tmp-dir)/bin/tests-tmux-autocomplete

    coproc:get-stdin-fd $ta stdin
    # doesn't work without these calls, the process will not start.
    coproc:get-stdout-fd $ta stdout
    coproc:get-stderr-fd $ta stdout
    coproc:close-fd stdin

    # no idea how to catch running process
    sleep 0.2
}

:tmux-complete-wait() {
    coproc:wait $ta
}
