module github.com/kubernetes-csi/csi-driver-smb

go 1.24

toolchain go1.24.0

godebug winsymlink=0

require (
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/adal v0.9.23
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/container-storage-interface/spec v1.11.0
	github.com/golang/protobuf v1.5.4
	github.com/kubernetes-csi/csi-lib-utils v0.13.0
	github.com/kubernetes-csi/csi-proxy/client v1.0.1
	github.com/onsi/ginkgo/v2 v2.17.1
	github.com/onsi/gomega v1.33.0
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.7.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/net v0.35.0
	google.golang.org/grpc v1.65.0
	k8s.io/api v0.28.12
	k8s.io/apimachinery v0.28.12
	k8s.io/client-go v0.28.12
	k8s.io/component-base v0.28.12
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubernetes v1.28.12
	k8s.io/mount-utils v0.32.0
	k8s.io/pod-security-admission v0.28.8
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738
	sigs.k8s.io/cloud-provider-azure v1.28.9
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/cel-go v0.16.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/selinux v1.10.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/v3 v3.5.9 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0 // indirect
	go.opentelemetry.io/otel v1.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.20.0 // indirect
	go.opentelemetry.io/otel/sdk v1.20.0 // indirect
	go.opentelemetry.io/otel/trace v1.20.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.19.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/oauth2 v0.20.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	google.golang.org/genproto v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240709173604-40e1e62336c5 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.0.0 // indirect
	k8s.io/apiserver v0.28.12 // indirect
	k8s.io/cloud-provider v0.28.12 // indirect
	k8s.io/component-helpers v0.28.12 // indirect
	k8s.io/controller-manager v0.28.12 // indirect
	k8s.io/kms v0.28.12 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	k8s.io/kubectl v0.0.0 // indirect
	k8s.io/kubelet v0.28.12 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.1.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.28.12
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.28.12
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.28.12
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.28.12
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.28.12
	k8s.io/endpointslice => k8s.io/endpointslice v0.28.12
	k8s.io/gengo => k8s.io/gengo v0.0.0-20200114144118-36b2048a9120
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.28.12
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.28.12
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.28.12
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.28.12
	k8s.io/kubectl => k8s.io/kubectl v0.28.12
	k8s.io/kubelet => k8s.io/kubelet v0.28.12
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.28.12
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.28.12
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.3.0

replace github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.3.1

replace github.com/Azure/go-autorest/logger => github.com/Azure/go-autorest/logger v0.2.1

replace github.com/Azure/go-autorest/tracing => github.com/Azure/go-autorest/tracing v0.6.0

replace github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.6.0

replace github.com/NYTimes/gziphandler => github.com/NYTimes/gziphandler v1.1.1

replace github.com/antlr/antlr4/runtime/Go/antlr/v4 => github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df

replace github.com/asaskevich/govalidator => github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a

replace github.com/beorn7/perks => github.com/beorn7/perks v1.0.1

replace github.com/blang/semver/v4 => github.com/blang/semver/v4 v4.0.0

replace github.com/cenkalti/backoff/v4 => github.com/cenkalti/backoff/v4 v4.2.1

replace github.com/cespare/xxhash/v2 => github.com/cespare/xxhash/v2 v2.3.0

replace github.com/coreos/go-semver => github.com/coreos/go-semver v0.3.1

replace github.com/coreos/go-systemd/v22 => github.com/coreos/go-systemd/v22 v22.5.0

replace github.com/davecgh/go-spew => github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc

replace github.com/docker/distribution => github.com/docker/distribution v2.8.2+incompatible

replace github.com/emicklei/go-restful/v3 => github.com/emicklei/go-restful/v3 v3.12.1

replace github.com/evanphx/json-patch => github.com/evanphx/json-patch v5.9.0+incompatible

replace github.com/felixge/httpsnoop => github.com/felixge/httpsnoop v1.0.4

replace github.com/fsnotify/fsnotify => github.com/fsnotify/fsnotify v1.7.0

replace github.com/go-logr/logr => github.com/go-logr/logr v1.4.2

replace github.com/go-logr/stdr => github.com/go-logr/stdr v1.2.2

replace github.com/go-openapi/jsonpointer => github.com/go-openapi/jsonpointer v0.19.6

replace github.com/go-openapi/jsonreference => github.com/go-openapi/jsonreference v0.20.2

replace github.com/go-openapi/swag => github.com/go-openapi/swag v0.22.3

replace github.com/go-task/slim-sprig => github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace github.com/golang-jwt/jwt/v4 => github.com/golang-jwt/jwt/v4 v4.5.0

replace github.com/golang/groupcache => github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da

replace github.com/google/cel-go => github.com/google/cel-go v0.16.1

replace github.com/google/gnostic-models => github.com/google/gnostic-models v0.6.8

replace github.com/google/go-cmp => github.com/google/go-cmp v0.6.0

replace github.com/google/gofuzz => github.com/google/gofuzz v1.2.0

replace github.com/google/pprof => github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1

replace github.com/google/uuid => github.com/google/uuid v1.6.0

replace github.com/grpc-ecosystem/go-grpc-prometheus => github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0

replace github.com/grpc-ecosystem/grpc-gateway/v2 => github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.6

replace github.com/inconshreveable/mousetrap => github.com/inconshreveable/mousetrap v1.1.0

replace github.com/josharian/intern => github.com/josharian/intern v1.0.0

replace github.com/json-iterator/go => github.com/json-iterator/go v1.1.12

replace github.com/mailru/easyjson => github.com/mailru/easyjson v0.7.7

replace github.com/matttproud/golang_protobuf_extensions => github.com/matttproud/golang_protobuf_extensions v1.0.4

replace github.com/moby/spdystream => github.com/moby/spdystream v0.2.0

replace github.com/moby/sys/mountinfo => github.com/moby/sys/mountinfo v0.7.2

replace github.com/moby/sys/userns => github.com/moby/sys/userns v0.1.0

replace github.com/modern-go/concurrent => github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd

replace github.com/modern-go/reflect2 => github.com/modern-go/reflect2 v1.0.2

replace github.com/munnerz/goautoneg => github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822

replace github.com/opencontainers/go-digest => github.com/opencontainers/go-digest v1.0.0

replace github.com/opencontainers/selinux => github.com/opencontainers/selinux v1.10.0

replace github.com/pkg/errors => github.com/pkg/errors v0.9.1

replace github.com/pmezard/go-difflib => github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.16.0

replace github.com/prometheus/client_model => github.com/prometheus/client_model v0.4.0

replace github.com/prometheus/common => github.com/prometheus/common v0.44.0

replace github.com/prometheus/procfs => github.com/prometheus/procfs v0.10.1

replace github.com/spf13/cobra => github.com/spf13/cobra v1.8.0

replace github.com/spf13/pflag => github.com/spf13/pflag v1.0.5

replace github.com/stoewer/go-strcase => github.com/stoewer/go-strcase v1.2.0

replace go.etcd.io/etcd/api/v3 => go.etcd.io/etcd/api/v3 v3.5.9

replace go.etcd.io/etcd/client/pkg/v3 => go.etcd.io/etcd/client/pkg/v3 v3.5.9

replace go.etcd.io/etcd/client/v3 => go.etcd.io/etcd/client/v3 v3.5.9

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.0

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0

replace go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.20.0

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace => go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.20.0

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.20.0

replace go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.20.0

replace go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.20.0

replace go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.20.0

replace go.opentelemetry.io/proto/otlp => go.opentelemetry.io/proto/otlp v1.0.0

replace go.uber.org/atomic => go.uber.org/atomic v1.10.0

replace go.uber.org/multierr => go.uber.org/multierr v1.11.0

replace go.uber.org/zap => go.uber.org/zap v1.19.0

replace golang.org/x/crypto => golang.org/x/crypto v0.33.0

replace golang.org/x/exp => golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56

replace golang.org/x/mod => golang.org/x/mod v0.19.0

replace golang.org/x/oauth2 => golang.org/x/oauth2 v0.20.0

replace golang.org/x/sync => golang.org/x/sync v0.11.0

replace golang.org/x/sys => golang.org/x/sys v0.30.0

replace golang.org/x/term => golang.org/x/term v0.29.0

replace golang.org/x/text => golang.org/x/text v0.22.0

replace golang.org/x/time => golang.org/x/time v0.3.0

replace golang.org/x/tools => golang.org/x/tools v0.23.0

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20231016165738-49dd2c1f3d0b

replace google.golang.org/genproto/googleapis/api => google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157

replace google.golang.org/genproto/googleapis/rpc => google.golang.org/genproto/googleapis/rpc v0.0.0-20240709173604-40e1e62336c5

replace google.golang.org/protobuf => google.golang.org/protobuf v1.34.2

replace gopkg.in/inf.v0 => gopkg.in/inf.v0 v0.9.1

replace gopkg.in/natefinch/lumberjack.v2 => gopkg.in/natefinch/lumberjack.v2 v2.2.1

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0

replace gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1

replace k8s.io/apiserver => k8s.io/apiserver v0.28.12

replace k8s.io/component-helpers => k8s.io/component-helpers v0.28.12

replace k8s.io/controller-manager => k8s.io/controller-manager v0.28.12

replace k8s.io/kms => k8s.io/kms v0.28.12

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9

replace sigs.k8s.io/apiserver-network-proxy/konnectivity-client => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.1.2

replace sigs.k8s.io/json => sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd

replace sigs.k8s.io/structured-merge-diff/v4 => sigs.k8s.io/structured-merge-diff/v4 v4.2.3

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v68.0.0+incompatible

replace github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.11.29

replace github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.9.23

replace github.com/Azure/go-autorest/autorest/to => github.com/Azure/go-autorest/autorest/to v0.4.0

replace github.com/container-storage-interface/spec => github.com/container-storage-interface/spec v1.11.0

replace github.com/golang/protobuf => github.com/golang/protobuf v1.5.4

replace github.com/kubernetes-csi/csi-lib-utils => github.com/kubernetes-csi/csi-lib-utils v0.13.0

replace github.com/kubernetes-csi/csi-proxy/client => github.com/kubernetes-csi/csi-proxy/client v1.0.1

replace github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.17.1

replace github.com/onsi/gomega => github.com/onsi/gomega v1.33.0

replace github.com/pborman/uuid => github.com/pborman/uuid v1.2.1

replace github.com/pelletier/go-toml => github.com/pelletier/go-toml v1.7.0

replace github.com/stretchr/testify => github.com/stretchr/testify v1.9.0

replace golang.org/x/net => golang.org/x/net v0.35.0

replace google.golang.org/grpc => google.golang.org/grpc v1.65.0

replace k8s.io/api => k8s.io/api v0.28.12

replace k8s.io/apimachinery => k8s.io/apimachinery v0.28.12

replace k8s.io/client-go => k8s.io/client-go v0.28.12

replace k8s.io/component-base => k8s.io/component-base v0.28.12

replace k8s.io/klog/v2 => k8s.io/klog/v2 v2.130.1

replace k8s.io/kubernetes => k8s.io/kubernetes v1.28.12

replace k8s.io/mount-utils => k8s.io/mount-utils v0.32.0

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.28.8

replace k8s.io/utils => k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738

replace sigs.k8s.io/cloud-provider-azure => sigs.k8s.io/cloud-provider-azure v1.28.9

replace sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.3.0
