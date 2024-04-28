package main

import (
	"bytetrade.io/web3os/tapr/cmd/upload/app"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)

	cmd := &cobra.Command{
		Use:   "upload-gateway",
		Short: "upload gateway",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			server := &app.Server{}

			err := server.Init()
			if err != nil {
				klog.Fatalln(err)
				panic(err)
			}

			server.ServerRun()

			klog.Info("upload shutdown ")
		},
	}

	klog.Info("upload starting ... ")

	if err := cmd.Execute(); err != nil {
		klog.Fatalln(err)
	}

}
