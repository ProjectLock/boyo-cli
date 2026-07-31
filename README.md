# Boyo (`boyo-cli`) 🏴󠁧󠁢󠁷󠁬󠁳󠁿

> Your friendly, no-nonsense cloud infrastructure helper.

**Boyo** is a light-weight Go CLI tool designed to rapidly prototype cloud infrastructure. Instead of manually configuring cloud resources or wrestling with heavy configuration files for simple tests, Boyo uses pre-baked templates to spin up production-ready cloud environments in seconds.

---

## 📋 Prerequisites & Versions

* **Go Version:** `go1.26.4 darwin/arm64` (or Go 1.22+)
* **AWS Credentials:** Active AWS CLI credentials configured locally (`~/.aws/credentials` or environment variables like `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`).

---

## 🚀 Quick Start

### 1. Build the Binary

Clone the repository and build the local binary:

```bash
git clone [https://github.com/ProjectLock/boyo-cli.git](https://github.com/ProjectLock/boyo-cli.git)
cd boyo-cli

# Compile the binary as 'boyo'
go build -o boyo