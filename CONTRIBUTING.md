# Contributing to GoRL

Thank you for considering a contribution to GoRL! To ensure a smooth and productive collaboration, please follow the guidelines below.

---

## Code of Conduct

We expect all contributors to adhere to our [Code of Conduct](./CODE_OF_CONDUCT.md). In summary:

* **Be respectful**: Treat others with courtesy and kindness.
* **Be collaborative**: Offer constructive feedback and welcome diverse perspectives.
* **Be professional**: Focus discussions on technical aspects and project goals.

## Reporting Issues

1. **Search existing issues** to avoid duplicates.
2. If none match, open a new issue and select the appropriate template:

   * **Bug Report**: Provide steps to reproduce, expected vs. actual behavior, and environment details.
   * **Feature Request**: Describe the desired feature, use cases, and any design ideas.
3. Use clear, concise language and include code snippets, log outputs, or configuration details as needed.

## Branching Strategy

* **`main`**: Always in a deployable state.
* **Feature branches**: `feature/<short-description>`, e.g., `feature/sliding-window-tuning`.
* **Bugfix branches**: `bugfix/<short-description>`, e.g., `bugfix/redis-expiry`.
* **Documentation branches**: `docs/<short-description>`, e.g., `docs/api-improvements`.

## Development Workflow

1. **Fork** the repository on GitHub.

2. **Clone** your fork and create a new branch:

   ```bash
   git clone https://github.com/AliRizaAynaci/gorl.git
   cd gorl
   git checkout -b feature/your-feature
   ```

3. **Implement** your changes, ensuring you follow the style guidelines below.

4. **Test** thoroughly:

   ```bash
   go test ./... -race -cover
   ```

5. **Commit** with a descriptive message (see Conventional Commits below).

6. **Push** your branch:

   ```bash
   git push origin feature/your-feature
   ```

7. **Open a Pull Request** against `main`, choose the appropriate template, and provide context for reviewers.

## Style Guidelines

* **Formatting**: Run `go fmt ./...` before committing.
* **Linting**: Address lint warnings from `golangci-lint run`.
* **Imports**: Group by standard library, external modules, then internal packages.
* **Documentation**: Document all exported types and functions with GoDoc comments.

## Testing

* Strive for high coverage, especially in core packages.
* Prefer table-driven tests for clarity and maintainability.
* Mock external dependencies (e.g., Redis) or document integration test requirements clearly.

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org):

```
<type>(scope?): <subject>

<body>

<footer>
```

**Common types**:

* `feat`: New feature
* `fix`: Bug fix
* `docs`: Documentation only
* `style`: Formatting, no code change
* `refactor`: Code change without adding features or fixing bugs
* `test`: Adding or updating tests
* `chore`: Maintenance tasks, build and tooling changes

## Pull Request Process

1. Assign 1–2 reviewers.
2. Ensure all CI checks pass (tests, lint, formatting).
3. Address review comments with prompt, focused updates.
4. Once approved, a maintainer will merge your PR.
