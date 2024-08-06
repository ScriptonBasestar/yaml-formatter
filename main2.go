package main

import (
    "fmt"
    "io/ioutil"

    "gopkg.in/yaml.v2"
)

type Rule struct {
    Version    string                 `yaml:"version,omitempty"`
    Services   map[string]interface{} `yaml:"services"`
    Volumes    map[string]interface{} `yaml:"volumes,omitempty"`
    Networks   map[string]interface{} `yaml:"networks,omitempty"`
    Secrets    map[string]interface{} `yaml:"secrets,omitempty"`
    NonSort    map[string]interface{} `yaml:"non_sort,omitempty"`
}

type ComposeFile struct {
    Services map[string]Service `yaml:"services"`
}

type Service struct {
    Image       string              `yaml:"image,omitempty"`
    Restart     string              `yaml:"restart,omitempty"`
    Ports       []string            `yaml:"ports,omitempty"`
    Volumes     []string            `yaml:"volumes,omitempty"`
    Environment []string            `yaml:"environment,omitempty"`
    Profiles    []string            `yaml:"profiles,omitempty"`
}

func main() {
    // rule.yaml 파일 읽기
    ruleData, err := ioutil.ReadFile("rules/docker-compose.rule.yaml")
    if err != nil {
        fmt.Printf("Error reading rule.yaml: %v\n", err)
        return
    }

    var rule Rule
    err = yaml.Unmarshal(ruleData, &rule)
    if err != nil {
        fmt.Printf("Error unmarshaling rule.yaml: %v\n", err)
        return
    }

    // docker-compose.yaml 파일 읽기
    composeData, err := ioutil.ReadFile("test/docker-compose.yaml")
    if err != nil {
        fmt.Printf("Error reading docker-compose.yaml: %v\n", err)
        return
    }

    var composeFile ComposeFile
    err = yaml.Unmarshal(composeData, &composeFile)
    if err != nil {
        fmt.Printf("Error unmarshaling docker-compose.yaml: %v\n", err)
        return
    }

    // 정렬된 결과를 저장할 맵
    sortedServices := make(map[string]interface{})

    // rule.yaml의 서비스 순서에 맞춰 정렬
    for _, serviceKey := range []string{"redis", "postgres"} { // 여기서 서비스 순서를 명시합니다.
        if service, exists := composeFile.Services[serviceKey]; exists {
            sortedServices[serviceKey] = service
        }
    }

    // 정렬된 docker-compose.yaml 출력
    output := make(map[string]interface{})
    output["services"] = sortedServices

    // YAML 포맷으로 변환
    sortedOutput, err := yaml.Marshal(output)
    if err != nil {
        fmt.Printf("Error marshaling sorted output: %v\n", err)
        return
    }

    // 결과 파일에 쓰기
    err = ioutil.WriteFile("test/sorted-docker-compose.yaml", sortedOutput, 0644)
    if err != nil {
        fmt.Printf("Error writing to sorted-docker-compose.yaml: %v\n", err)
        return
    }

    fmt.Println("docker-compose.yaml가 rule.yaml에 기반하여 정렬되었습니다!")
}
