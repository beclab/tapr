package nats

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/mitchellh/mapstructure"
	load "github.com/nats-io/nats-server/conf"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const tmpl = `{
  "http_port": {{.HTTPPort}},
  "jetstream": {
    "max_file_store": {{.Jetstream.MaxFileStore}},
    "max_memory_store": {{.Jetstream.MaxMemoryStore}},
    "store_dir": "{{.Jetstream.StoreDir}}"
  },
  "accounts": {
    "terminus": {
      "jetstream": enabled,
      "users": [
        {{- range $index, $user := .Accounts.Terminus.Users }}
        {{- if $index}},{{ end }}
        {
          "user": "{{ $user.Username }}",
          {{ if eq $user.Username "admin" }}
          "password": $ADMIN_PASSWORD,
          {{ else }}
          "password": {{ $user.Password | quoteOrNot}},
          {{ end }}
          "permissions": {
            "publish": {
              "allow": [{{ range $i, $allow := $user.Permissions.Publish.Allow }}{{ if $i }}, {{ end }}"{{ $allow }}"{{ end }}]
            },
            "subscribe": {
              "allow": [{{ range $i, $allow := $user.Permissions.Subscribe.Allow }}{{ if $i }}, {{ end }}"{{ $allow }}"{{ end }}]
            }
          }
        }
        {{- end }}
      ]
    }
  },
  "port": {{ .Port }},
  "pid_file": "{{ .PidFile }}"
  "server_name": "{{ .ServerName }}"
}`

func quoteOrNot(s string) string {
	if strings.HasPrefix(s, "$2a") {
		return s
	}
	if len(s) > 0 && s[0] == '$' {
		return s
	}
	return `"` + s + `"`
}

func renderConfigFile(config *Config) ([]byte, error) {
	funcMap := template.FuncMap{
		"quoteOrNot": quoteOrNot,
	}
	klog.Infof("renderConfigFile: %##v\n", config)
	var buf bytes.Buffer
	tpl := template.Must(template.New("config").Funcs(funcMap).Parse(tmpl))
	err := tpl.Execute(&buf, config)
	if err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

func RenderConfigFile(config *Config) error {
	data, err := renderConfigFile(config)
	if err != nil {
		return err
	}
	clientSet, err := newClientSet()
	if err != nil {
		return err
	}
	cm, err := clientSet.CoreV1().ConfigMaps("os-system").Get(context.TODO(), "nats-config", metav1.GetOptions{})
	if err != nil {
		return err
	}
	cm.Data["nats.conf"] = string(data)
	_, err = clientSet.CoreV1().ConfigMaps("os-system").Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func ParseFile(fp string) (*Config, error) {
	m, err := load.ParseFile(fp)
	if err != nil {
		return nil, err
	}
	var config Config
	err = mapstructure.Decode(m, &config)
	if err != nil {
		klog.Infof("mapstructure decode: err=%v", err)
		return nil, err
	}
	return &config, nil
}
