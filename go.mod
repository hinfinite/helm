module github.com/hinfinite/helm

go 1.13

require (
	github.com/Azure/go-autorest/autorest v0.11.20 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.15 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/Masterminds/vcs v1.13.1
	github.com/Microsoft/go-winio v0.5.1 // indirect
	// github.com/Microsoft/hcsshim v0.9.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496
	github.com/bshuster-repo/logrus-logstash-hook v1.0.0 // indirect
	github.com/containerd/cgroups v1.0.2 // indirect
	github.com/containerd/containerd v1.5.9
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/deislabs/oras v0.8.1
	github.com/docker/cli v20.10.11+incompatible // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.11+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/flock v0.7.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/mattn/go-shellwords v1.0.10
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/copystructure v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/xeipuuv/gojsonschema v1.1.0
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871
	golang.org/x/net v0.0.0-20220107192237-5cfca573fb4d // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	google.golang.org/grpc v1.43.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/v3 v3.0.2 // indirect
	helm.sh/helm/v3 v3.1.2
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/cli-runtime v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.17.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/BurntSushi/toml => github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver/v3 => github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 => github.com/Masterminds/sprig/v3 v3.2.2
	github.com/Masterminds/vcs => github.com/Masterminds/vcs v1.13.1
	github.com/asaskevich/govalidator => github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.2
	github.com/cyphar/filepath-securejoin => github.com/cyphar/filepath-securejoin v0.2.2
	github.com/deislabs/oras => github.com/deislabs/oras v0.8.1
	// github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-units => github.com/docker/go-units v0.4.0
	github.com/evanphx/json-patch => github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/gobwas/glob => github.com/gobwas/glob v0.2.3
	github.com/gofrs/flock => github.com/gofrs/flock v0.7.1
	github.com/golang/glog => github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gosuri/uitable => github.com/gosuri/uitable v0.0.4
	github.com/mattn/go-shellwords => github.com/mattn/go-shellwords v1.0.10
	github.com/mitchellh/copystructure => github.com/mitchellh/copystructure v1.0.0
	github.com/opencontainers/go-digest => github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.1
	github.com/patrickmn/go-cache => github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors => github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus => github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra => github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify => github.com/stretchr/testify v1.5.1
	github.com/xeipuuv/gojsonschema => github.com/xeipuuv/gojsonschema v1.1.0
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.1.2
	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.3
	k8s.io/client-go => k8s.io/client-go v0.17.3
	k8s.io/klog => k8s.io/klog v1.0.0
	k8s.io/kubectl => k8s.io/kubectl v0.17.3
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.1.0
)
