package main

import (
	"fmt"
)

const (
	intro = `Hello! Thank you for trying out tmux-autocomplete.

tmux-autocomplete should not be running from terminal, it should be running
using tmux bindings, put following line to your ~/.tmux.conf:

  bind-key C-Space run -b 'tmux-autocomplete'

Then reload tmux configuration using following command:

  tmux source ~/.tmux.conf

Now tmux-autocomplete can be triggered using following key bindings:

  Control+b Control+Space

You can change default prefix key binding Control+b to Control+Space too, add
following line to your ~/.tmux.conf and then reload tmux configuration again:

  set -g prefix C-Space

Now tmux-autocomplete can be triggered using following key bindings:
  Control+Space Control+Space

If you have any questions, don't hesitate to contact us:
  https://tmux.reconquest.io/support/
`
)

func printIntroductionMessage() {
	fmt.Println(intro)
}
