package resources

import "strings"

type Contexts struct{}

func (c *Contexts) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 30},
		{Name: "STATUS", Width: 13},
		{Name: "CLUSTER", Width: 28},
		{Name: "SERVER", Width: 50},
		{Name: "AGE", Width: 6},
	}
}

func (c *Contexts) TableRow(item ResourceItem) []string {
	cluster := item.Name
	server := "https://127.0.0.1:6443"
	switch item.Name {
	case "dev-us-east-1":
		cluster = "eks-dev-us-east-1"
		server = "https://ABCDEF1234.gr7.us-east-1.eks.amazonaws.com"
	case "staging-eu-west-1":
		cluster = "eks-staging-eu-west-1"
		server = "https://GHIJKL5678.gr7.eu-west-1.eks.amazonaws.com"
	case "prod-us-east-1":
		cluster = "eks-prod-us-east-1"
		server = "https://MNOPQR9012.gr7.us-east-1.eks.amazonaws.com"
	case "prod-eu-west-1":
		cluster = "eks-prod-eu-west-1"
		server = "https://STUVWX3456.gr7.eu-west-1.eks.amazonaws.com"
	case "minikube":
		cluster = "minikube"
		server = "https://192.168.49.2:8443"
	case "docker-desktop":
		cluster = "docker-desktop"
		server = "https://127.0.0.1:6443"
	}
	return []string{item.Name, item.Status, cluster, server, item.Age}
}

func NewContexts() *Contexts {
	return &Contexts{}
}

func (c *Contexts) Name() string { return "contexts" }
func (c *Contexts) Key() rune   { return 'X' }

func (c *Contexts) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "dev-us-east-1", Status: "Available", Age: "120d"},
		{Name: "staging-eu-west-1", Status: "Available", Age: "90d"},
		{Name: "prod-us-east-1", Status: "Available", Age: "180d"},
		{Name: "prod-eu-west-1", Status: "Available", Age: "180d"},
		{Name: "minikube", Status: "Available", Age: "30d"},
		{Name: "docker-desktop", Status: "Unreachable", Age: "60d"},
	}
	c.Sort(items)
	return items
}

func (c *Contexts) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (c *Contexts) Detail(item ResourceItem) DetailData {
	cluster := item.Name
	user := "admin"
	server := "https://127.0.0.1:6443"

	switch item.Name {
	case "dev-us-east-1":
		cluster = "eks-dev-us-east-1"
		user = "arn:aws:eks:us-east-1:123456789012:cluster/dev"
		server = "https://ABCDEF1234.gr7.us-east-1.eks.amazonaws.com"
	case "staging-eu-west-1":
		cluster = "eks-staging-eu-west-1"
		user = "arn:aws:eks:eu-west-1:123456789012:cluster/staging"
		server = "https://GHIJKL5678.gr7.eu-west-1.eks.amazonaws.com"
	case "prod-us-east-1":
		cluster = "eks-prod-us-east-1"
		user = "arn:aws:eks:us-east-1:123456789012:cluster/prod"
		server = "https://MNOPQR9012.gr7.us-east-1.eks.amazonaws.com"
	case "prod-eu-west-1":
		cluster = "eks-prod-eu-west-1"
		user = "arn:aws:eks:eu-west-1:123456789012:cluster/prod-eu"
		server = "https://STUVWX3456.gr7.eu-west-1.eks.amazonaws.com"
	case "minikube":
		cluster = "minikube"
		user = "minikube"
		server = "https://192.168.49.2:8443"
	case "docker-desktop":
		cluster = "docker-desktop"
		user = "docker-desktop"
		server = "https://127.0.0.1:6443"
	}

	return DetailData{
		StatusLine: item.Status + "    cluster: " + cluster + "    server: " + server,
		Events:     []string{"—   Contexts do not produce events"},
		Labels: []string{
			"cluster=" + cluster,
			"user=" + user,
		},
	}
}

func (c *Contexts) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for contexts.",
	}
}

func (c *Contexts) Events(item ResourceItem) []string {
	return []string{"—   Contexts do not produce events"}
}

func (c *Contexts) Describe(item ResourceItem) string {
	return "Name:      " + item.Name + "\n" +
		"Status:    " + item.Status + "\n" +
		"Age:       " + item.Age
}

func (c *Contexts) YAML(item ResourceItem) string {
	cluster := item.Name
	user := "admin"
	server := "https://127.0.0.1:6443"
	ns := "default"

	switch item.Name {
	case "dev-us-east-1":
		cluster = "eks-dev-us-east-1"
		user = "arn:aws:eks:us-east-1:123456789012:cluster/dev"
		server = "https://ABCDEF1234.gr7.us-east-1.eks.amazonaws.com"
		ns = "dev"
	case "staging-eu-west-1":
		cluster = "eks-staging-eu-west-1"
		user = "arn:aws:eks:eu-west-1:123456789012:cluster/staging"
		server = "https://GHIJKL5678.gr7.eu-west-1.eks.amazonaws.com"
		ns = "staging"
	case "prod-us-east-1":
		cluster = "eks-prod-us-east-1"
		user = "arn:aws:eks:us-east-1:123456789012:cluster/prod"
		server = "https://MNOPQR9012.gr7.us-east-1.eks.amazonaws.com"
		ns = "production"
	case "prod-eu-west-1":
		cluster = "eks-prod-eu-west-1"
		user = "arn:aws:eks:eu-west-1:123456789012:cluster/prod-eu"
		server = "https://STUVWX3456.gr7.eu-west-1.eks.amazonaws.com"
		ns = "production"
	case "minikube":
		cluster = "minikube"
		user = "minikube"
		server = "https://192.168.49.2:8443"
	case "docker-desktop":
		cluster = "docker-desktop"
		user = "docker-desktop"
		server = "https://127.0.0.1:6443"
	}

	return strings.TrimSpace(`apiVersion: v1
kind: Config
preferences: {}
current-context: ` + item.Name + `
contexts:
- context:
    cluster: ` + cluster + `
    namespace: ` + ns + `
    user: ` + user + `
  name: ` + item.Name + `
clusters:
- cluster:
    certificate-authority-data: <redacted>
    server: ` + server + `
  name: ` + cluster + `
users:
- name: ` + user + `
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
      - eks
      - get-token
      - --cluster-name
      - ` + cluster)
}
