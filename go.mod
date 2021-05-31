module sigs.k8s.io/cluster-api-provider-linode

go 1.16

require (
	github.com/digitalocean/godo v1.54.0
	github.com/go-logr/logr v0.4.0
	github.com/miekg/dns v1.1.3
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20201203001011-0b49973bad19
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/klog/v2 v2.8.0
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10
	sigs.k8s.io/cluster-api v0.3.11-0.20210528213424-a74b6a6428cb
	sigs.k8s.io/controller-runtime v0.9.0-beta.5
)
