package genimports

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ModulePrefix = "github.com/clawscli/claws"
	CustomDir    = "custom"
)

func FindRegisterPackages(projectRoot string) ([]string, error) {
	var packages []string

	customDir := filepath.Join(projectRoot, CustomDir)

	err := filepath.Walk(customDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "register.go" && !info.IsDir() {
			dir := filepath.Dir(path)
			relPath, err := filepath.Rel(projectRoot, dir)
			if err != nil {
				return err
			}
			importPath := ModulePrefix + "/" + filepath.ToSlash(relPath)
			packages = append(packages, importPath)
		}

		return nil
	})

	sort.Strings(packages)
	return packages, err
}

func GetProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(wd, "cmd/claws") {
		return filepath.Join(wd, "..", ".."), nil
	}

	return wd, nil
}

var ServiceDisplayNames = map[string]string{
	"accessanalyzer":    "Access Analyzer",
	"acm":               "ACM",
	"apigateway":        "API Gateway",
	"apprunner":         "App Runner",
	"appsync":           "AppSync",
	"athena":            "Athena",
	"autoscaling":       "Auto Scaling",
	"backup":            "AWS Backup",
	"batch":             "Batch",
	"bedrock":           "Bedrock",
	"bedrock-agent":     "Bedrock Agent",
	"bedrock-agentcore": "Bedrock AgentCore",
	"budgets":           "Budgets",
	"ce":                "Cost Explorer",
	"cfn":               "CloudFormation",
	"cloudfront":        "CloudFront",
	"cloudtrail":        "CloudTrail",
	"cloudwatch":        "CloudWatch",
	"codebuild":         "CodeBuild",
	"codepipeline":      "CodePipeline",
	"cognito-idp":       "Cognito",
	"compute-optimizer": "Compute Optimizer",
	"configservice":     "Config",
	"datasync":          "DataSync",
	"detective":         "Detective",
	"directconnect":     "Direct Connect",
	"dynamodb":          "DynamoDB",
	"ec2":               "EC2",
	"ecr":               "ECR",
	"ecs":               "ECS",
	"elasticache":       "ElastiCache",
	"elbv2":             "ELBv2 (ALB/NLB/GLB)",
	"emr":               "EMR",
	"events":            "EventBridge",
	"fms":               "Firewall Manager",
	"glue":              "Glue",
	"guardduty":         "GuardDuty",
	"health":            "Health",
	"iam":               "IAM",
	"inspector2":        "Inspector",
	"kinesis":           "Kinesis",
	"kms":               "KMS",
	"lambda":            "Lambda",
	"license-manager":   "License Manager",
	"macie2":            "Macie",
	"network-firewall":  "Network Firewall",
	"opensearch":        "OpenSearch",
	"organizations":     "Organizations",
	"rds":               "RDS",
	"redshift":          "Redshift",
	"risp":              "RI/SP (Reserved Instances, Savings Plans)",
	"route53":           "Route53",
	"s3":                "S3",
	"s3vectors":         "S3 Vectors",
	"sagemaker":         "SageMaker",
	"secretsmanager":    "Secrets Manager",
	"securityhub":       "Security Hub",
	"service-quotas":    "Service Quotas",
	"sns":               "SNS",
	"sqs":               "SQS",
	"ssm":               "SSM",
	"stepfunctions":     "Step Functions",
	"transcribe":        "Transcribe",
	"transfer":          "Transfer Family",
	"trustedadvisor":    "Trusted Advisor",
	"vpc":               "VPC",
	"wafv2":             "WAF",
	"xray":              "X-Ray",
}

func GetServiceDisplayName(service string) string {
	if name, ok := ServiceDisplayNames[service]; ok {
		return name
	}
	return strings.ToUpper(service[:1]) + service[1:]
}

func GroupByService(packages []string) map[string][]string {
	grouped := make(map[string][]string)

	prefix := ModulePrefix + "/" + CustomDir + "/"
	for _, pkg := range packages {
		rest := strings.TrimPrefix(pkg, prefix)
		parts := strings.SplitN(rest, "/", 2)
		service := parts[0]
		grouped[service] = append(grouped[service], pkg)
	}

	return grouped
}

// PackageInfo contains information about a resource package.
type PackageInfo struct {
	ImportPath  string
	Service     string
	Resource    string
	PackageName string
	DirPath     string
}

// GetPackageInfo extracts detailed information from an import path.
func GetPackageInfo(projectRoot, importPath string) PackageInfo {
	prefix := ModulePrefix + "/" + CustomDir + "/"
	rest := strings.TrimPrefix(importPath, prefix)
	parts := strings.SplitN(rest, "/", 2)

	service := parts[0]
	resource := ""
	if len(parts) > 1 {
		resource = parts[1]
	}

	dirPath := CustomDir + "/" + service + "/" + resource
	pkgName := readPackageName(filepath.Join(projectRoot, dirPath, "register.go"))
	if pkgName == "" {
		pkgName = strings.ReplaceAll(resource, "-", "")
		if idx := strings.LastIndex(pkgName, "/"); idx >= 0 {
			pkgName = pkgName[idx+1:]
		}
	}

	return PackageInfo{
		ImportPath:  importPath,
		Service:     service,
		Resource:    resource,
		PackageName: pkgName,
		DirPath:     dirPath,
	}
}

func readPackageName(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return strings.TrimPrefix(line, "package ")
		}
	}
	return ""
}

// GenerateConstantsFile generates the content for a constants.go file.
func GenerateConstantsFile(pkgName, service, resource string) []byte {
	return []byte(`// Code generated by go generate; DO NOT EDIT.
// To regenerate: task gen-imports

package ` + pkgName + `

// ServiceResourcePath is the canonical path for this resource type.
const ServiceResourcePath = "` + service + "/" + resource + `"
`)
}
