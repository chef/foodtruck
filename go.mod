module github.com/chef/foodtruck

go 1.15

replace github.com/fsnotify/fsnotify => github.com/sensu/fsnotify v1.4.8-0.20191126053121-0adce4777482

require (
	github.com/Azure/azure-amqp-common-go/v3 v3.0.0
	github.com/Azure/azure-event-hubs-go/v3 v3.3.0
	github.com/Azure/azure-sdk-for-go v46.3.0+incompatible
	github.com/Azure/go-amqp v0.12.8 // indirect
	github.com/Azure/go-autorest/autorest v0.11.6
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/afero v1.4.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
)
