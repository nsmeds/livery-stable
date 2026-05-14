# AGENTS.md

## Conventions

- You can suggest running git "add", "commit" or "push" commands when appropriate, but do not run them yourself. If you want to run git add, commit or push, ask for permission first.
- When writing summaries, aim for concise prose. Only use emojis when they help convey meaning in an organizational context. Do not include long blocks of code examples.
- Ask permission before writing summaries to disk.
- When adding a feature or any new piece of testable code, remember to add a unit test. If possible, start by writing the test first, then implement the feature to make the test pass.
- Unit tests should be isolated from each other.
- Unit tests should avoid mocks and stubs. If mocks and stubs seem needed, consider refactoring the code to be more testable instead - but ask permission before making any significant refactorings.
- Unit tests should test one unit of code at a time, and should not have side effects that impact other tests.
- Avoid making functions public just to test them. Instead, consider refactoring the code to allow testing without exposing internal functions.

