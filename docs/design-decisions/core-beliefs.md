# Core Beliefs

These beliefs guide our technical trade-offs.

1.  **AI-First**: optimizing for context retrieval by LLMs is more important than optimizing for human navigation in deep folder structures.
2.  **Correctness over Speed**: Specs must be updated before code.
3.  **Don't fix what isn't broken**: Working, stable code should not be refactored purely for aesthetics (e.g., DRY). The risk of breaking something and the cost of re-testing outweigh marginal improvements in code elegance — especially when the duplicated code is unlikely to change.
