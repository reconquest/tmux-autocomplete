#!/bin/bash

:tmux-start
:tmux-new
:tmux-sh print-data
:tmux-wait

:tmux-type "D"
:tmux-complete
:tmux-type "k"
:tmux-type "Enter"
:tmux-complete-wait

tests:eval :tmux-cat
tests:assert-stdout "@ D333"
