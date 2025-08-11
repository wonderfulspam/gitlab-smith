# GitLab Smith - TODO List

This file tracks the development tasks for the GitLab Smith project. Generated
by gemini-2.5-pro.

## High Priority

- [x] **Integrate Deployer with Validator:**
  - The `validator` should be able to use the `deployer` to spin up a local GitLab instance and run the "before" and "after" configurations to validate behavioral equivalence.
  - This will enable the "Full Behavioral Testing" mode described in the `pipeline-emulator-spec.md`.

- [x] **Enhance Renderer:**
  - Add support for generating a visual representation of the pipeline (e.g., a DOT graph or a Mermaid diagram).
  - This will make it much easier to understand the impact of refactoring changes on the pipeline structure.

- [x] **Refactor Analyzer:**
  - The `analyzer` package has a large number of check functions. These should be refactored for better organization and extensibility.
  - Consider grouping checks by category (e.g., `performance`, `security`) into sub-packages.
  - Explore using a more data-driven approach to define the checks, which would make it easier to add new ones.

## Medium Priority

- [ ] **Improve Differ's Improvement Pattern Detection:**
  - The heuristics for detecting improvement patterns in the `differ` can be made more robust and extensible.
  - Add support for detecting more complex refactoring patterns, such as the extraction of composite actions.

- [ ] **Implement Dependency Injection:**
  - Introduce a dependency injection framework (e.g., `wire`) to manage the dependencies between the different packages.
  - This will make the codebase more modular and easier to test.

- [ ] **Add More Test Scenarios:**
  - Add more refactoring scenarios to the `test/refactoring-scenarios` directory to cover a wider range of use cases.
  - Include scenarios that test the interaction between different GitLab CI features (e.g., `includes` and `extends`).

## Low Priority

- [ ] **Improve Documentation:**
  - Add more examples and tutorials to the `README.md` and `CLAUDE.md` files.
  - Create a more user-friendly version of the `pipeline-emulator-spec.md` for new contributors.

- [ ] **Add Command-Line Flags for Advanced Features:**
  - Add command-line flags to the `gitlab-smith` CLI to control advanced features, such as the output format of the `renderer` and the validation rules of the `validator`.
