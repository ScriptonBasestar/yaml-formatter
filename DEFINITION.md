# 📑 YAML Formatter – DEFINITION.md

> **요약**  
> 이 도구는 스키마(또는 프리셋)에 정의된 **키 순서**를 기준으로 YAML 파일을 자동 재배열합니다.  
> Go 언어로 작성되어 빠르고 단독 실행이 가능하며, CI·Git 훅·대용량 파일 처리에 최적화되어 있습니다.

---

## 1. 왜 필요한가? 🤔

YAML은 **키 순서를 보존**하는 포맷이지만,  
❶ _사람마다 머릿속에 “보기 편한” 순서가 다르며_  
❷ **개념적으로 묶여야 이해가 쉬운 구조**가 존재합니다.  

예를 들어 Docker Compose 파일을 작성할 때 우리는 보통

```yaml
services:
  app1:
    name:
    image:
    ...
networks:
```

처럼 **`services → networks`** 순으로 두 섹션을 나누길 원하지만,  
일반 포매터는 이를 존중하지 못해 가독성이 떨어집니다.  

> **YAML Formatter**는 _“내가 원하는 순서”_ 를 **스키마로 고정**해 주어  
> **협업 시 불필요한 리뷰 지적**과 **머지 충돌**을 줄여 줍니다.

---

## 2. 명령어 정의 🖥️

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

## 3. 주요 사용처 📂

- **Docker Compose** (`docker-compose.yml`)
- **Kubernetes** (`*.k8s.yaml`)
- **GitHub Actions** (`.github/workflows/*.yml`)
- **Ansible Playbook** (`playbook.yml`)
- **Helm Values** (`values.yaml`)

---

## 4. Pre-commit 훅 예시 🔗

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: sb-yaml-format
        name: sb-yaml-format
        entry: sb-yaml check
        language: system
        files: \.(yml|yaml)$
```

---

## 5. 라이선스 📝

본 프로젝트는 **MIT License**를 따릅니다. 기여를 환영합니다! 🙌
