package main

import "github.com/yoskeoka/ww/worktree"

func worktreeCreateOpts(glOpts *globalOpts) worktree.CreateOpts {
	return worktree.CreateOpts{
		DryRun: glOpts.dryRun,
	}
}
