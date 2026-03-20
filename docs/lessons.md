# Lessons Learned

- **Mistake**: I started investigating from the workspace root instead of the target child repository.
- **Pattern**: In this meta-repo, child projects live under the workspace root and task scope must be resolved before running repo-specific workflow or tests.
- **Rule**: When a request is about a child project, switch into that child repo first and perform branching, docs, specs, and verification there.
- **Applied**: All tasks in the `vibe-coding-workspace` meta-repo, especially requests involving `ww/`, `ai-arena/`, `reversi-adventure/`, or `vim-learning-game/`.
