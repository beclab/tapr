package watchers

import (
	"fmt"

	"github.com/coredns/corefile-migration/migration/corefile"
	"k8s.io/klog/v2"
)

func UpsertCorefile(data, userzone, ip string) (string, error) {
	file, err := corefile.New(data)
	if err != nil {
		klog.Error("parse corefile error, ", err)
		return "", err
	}

	if len(file.Servers) != 1 {
		klog.Warning("invalid corefile configuration")
		return data, nil
	}

	var newPlugins []*corefile.Plugin
	found := false
	newOptions := []*corefile.Option{
		{
			Name: "match",
			Args: []string{fmt.Sprintf("\\w*\\.?(%s\\.)$", userzone)},
		},
		{
			Name: "answer",
			Args: []string{"{{ .Name }}", "60", "IN", "A", ip},
		},
		{
			Name: "fallthrough",
			Args: []string{},
		},
	}
	userTemplateArgs := []string{"IN", "A", userzone}

	for _, p := range file.Servers[0].Plugins {
		// only care about template plugins
		if p.Name != "template" {
			newPlugins = append(newPlugins, p)
			continue
		}

		if len(p.Args) != 3 {
			// the template is not added by us, keep it
			klog.Info(p.Args)
			newPlugins = append(newPlugins, p)
			continue
		}

		if p.Args[2] == userTemplateArgs[2] {
			found = true
			p.Options = newOptions
			newPlugins = append(newPlugins, p)
		}
	}

	if !found {
		newPlugins = append(newPlugins, &corefile.Plugin{
			Name:    "template",
			Args:    userTemplateArgs,
			Options: newOptions,
		})
	}

	file.Servers[0].Plugins = newPlugins

	return file.ToString(), nil
}
