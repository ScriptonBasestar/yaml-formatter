# 📑 YAML Formatter

> **요약**  
> 이 도구는 스키마(또는 프리셋)에 정의된 **키 순서**를 기준으로 YAML 파일을 자동 재배열합니다.  
> Go 언어로 작성되어 빠르고 단독 실행이 가능하며, CI·Git 훅·대용량 파일 처리에 최적화되어 있습니다.
> 오버엔지니어링 없이 yaml을 재배열 및 포매팅 하는 간단한 빠르고 확실하게 제공합니다.

---

## 명령어 🖥️

```bash
# 스키마 생성
sb-yaml schema gen {schema_name} {yaml_file} > {out_schema_file}
sb-yaml schema gen compose docker-compose.yml

# 스키마 저장
sb-yaml schema set {schema_name} {schema_file}
sb-yaml schema set compose docker-compose.yml.schema

# YAML로부터 스키마 바로 저장
sb-yaml schema set {schema_name} --from-yaml {yaml_file}
sb-yaml schema set compose --from-yaml docker-compose.yml

# 스키마 목록 확인
sb-yaml schema list

# 포매팅 실행
sb-yaml format {schema_name} <files>
sb-yaml format compose docker-compose.yaml
sb-yaml format k8s *.k8s.yaml

# 포매팅 체크 전용(변경 여부만 확인)
sb-yaml check {schema_name} <files>
sb-yaml check compose docker-compose.yaml
sb-yaml check k8s *.k8s.yaml

# Git pre-commit 훅 스니펫 출력
sb-yaml show git-pre-commit-hook
```

---

## 주요 사용처 📂

- **Docker Compose** (`docker-compose.yml`)
- **Kubernetes** (`*.k8s.yaml`)
- **GitHub Actions** (`.github/workflows/*.yml`)
- **Ansible Playbook** (`playbook.yml`)
- **Helm Values** (`values.yaml`)

---

## Build & Test

```bash
# Build
go build -o sb-yaml .

# Test
go test ./...
```