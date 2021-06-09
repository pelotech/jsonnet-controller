local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';
local kubecfg = import 'internal://lib/kubecfg.libsonnet';

{
    local this = self,

    // The prefix to use for names of resources
    name_prefix:: 'kubecfg-operator',
    // The namespace to deploy resources in
    namespace:: 'kubecfg-system',
    // Whether to create the namespace
    create_namespace:: true,
    // Whether the cluster-admin role should be tied to the manager
    cluster_admin:: true,
    // If setting cluster_admin: false, fill out additional RBAC rules
    // you'd like to assign to the manager.
    additional_rules:: [],

    // Resource labels
    labels:: {
        app: this.name_prefix,
        control_plane: 'manager'
    },

    // Manager Deployment options
    manager_image:: 'ghcr.io/pelotech/kubecfg-controller:latest',
    manager_pull_policy:: 'IfNotPresent',
    manager_replicas:: 1,

    // Set to false to skip rendering CRDs
    install_crds:: true,

    // Enable flux support
    flux_enabled:: false,

    crds: if this.install_crds then [
        kubecfg.parseYaml(importstr '../crd/bases/apps.kubecfg.io_konfigurations.yaml'),
    ],

    control_namespace: if this.create_namespace then kube.Namespace(this.namespace) {
        metadata+: {
            labels: this.labels,
        },
    },

    rbac: {
        local rbac = self,
        local all_perms = ['create', 'delete', 'get', 'list', 'patch', 'update', 'watch'],
        local ro_perms = ['get', 'list', 'watch'],

        manager_service_account: kube.ServiceAccount(this.name_prefix + '-sa') {
            metadata+: {
                namespace: this.namespace,
                labels: this.labels,
            },
        },

        manager_role: kube.ClusterRole(this.name_prefix + '-manager-role') {
            metadata+: { labels: this.labels },
            rules: [
                {
                    apiGroups: ['apps.kubecfg.io'],
                    resources: ['konfigurations', 'konfigurations/finalizers', 'konfigurations/status'],
                    verbs: all_perms,
                },
                {
                    apiGroups: [''],
                    resources: ['secrets', 'serviceaccounts'],
                    verbs: ro_perms,
                },
                {
                    apiGroups: ['source.toolkit.fluxcd.io'],
                    resources: ['buckets', 'gitrepositories', 'buckets/status', 'gitrepositories/status'],
                    verbs: ro_perms,
                },
            ]
        },

        leader_election_role: kube.ClusterRole(this.name_prefix + '-leader-election-role') {
            metadata+: { labels: this.labels },
            rules: [
                {
                    apiGroups: [''],
                    resources: ['configmaps'],
                    verbs: all_perms
                },
                {
                    apiGroups: ['coordination.k8s.io'],
                    resources: ['leases'],
                    verbs: all_perms
                },
                {
                    apiGroups: [''],
                    resources: ['events'],
                    verbs: ['create', 'patch'],
                },
            ]
        },

        manage_role_binding: kube.ClusterRoleBinding(this.name_prefix + '-manager-role-binding') {
            metadata+: { labels: this.labels },
            subjects_:: [ rbac.manager_service_account ],
            roleRef_:: rbac.manager_role
        },

        leader_election_role_binding: kube.ClusterRoleBinding(this.name_prefix + '-leader-election-role-binding') {
            metadata+: { labels: this.labels },
            subjects_:: [ rbac.manager_service_account ],
            roleRef_:: rbac.leader_election_role
        },

        custom_role: if std.length(this.additional_rules) > 0 then kube.ClusterRole(this.name_prefix + '-manager-custom-role') {
            metadata+: { labels: this.labels },
            rules: this.additional_rules
        } else null,

        custom_role_binding: if self.custom_role != null then kube.ClusterRoleBinding(this.name_prefix + '-manager-custom-role-binding') {
            metadata+: { labels: this.labels },
            subjects_:: [ rbac.manager_service_account ],
            roleRef_:: this.custom_role
        } else null,

        cluster_admin_binding: if this.cluster_admin then kube.ClusterRoleBinding(this.name_prefix + '-cluster-admin-binding') {
            metadata+: { labels: this.labels },
            subjects_:: [ rbac.manager_service_account ],
            roleRef_:: { kind: 'ClusterRole', metadata: { name: 'cluster-admin' } }
        } else null
    },

    manager_deployment: kube.Deployment(this.name_prefix + '-manager') {
        metadata+: { 
            namespace: this.namespace,
            labels: this.labels
        },
        spec+: {
            replicas: this.manager_replicas,
            template+: {
                metadata+: {
                    labels+: this.labels,
                },
                spec+: {
                    serviceAccountName: this.rbac.manager_service_account.metadata.name,
                    securityContext: { runAsNonRoot: true },
                    terminationGracePeriodSeconds: 10,
                    volumes_: {
                        manager_cache: kube.EmptyDirVolume(),
                    }, 
                    containers_+: {
                        manager: kube.Container('manager') {
                            image: this.manager_image,
                            imagePullPolicy: this.manager_pull_policy,
                            command: ['/manager'],
                            args: [ '--leader-elect' ] + if this.flux_enabled then ['--flux-enabled'],
                            securityContext: { allowPrivilegeEscalation: false },
                            ports_+: {
                                http: { containerPort: 8080 },
                            },
                            volumeMounts_+: {
                                manager_cache: { mountPath: '/cache' },
                            },
                            livenessProbe: {
                                httpGet: { path: '/healthz', port: 8081 },
                                initialDelaySeconds: 15,
                                periodSeconds: 20
                            },
                            readinessProbe: {
                                httpGet: { path: '/readyz', port: 8081 },
                                initialDelaySeconds: 5,
                                periodSeconds: 10
                            },
                            resources: {
                                limits: { cpu: '100m', memory: '128Mi' },
                                requests: { cpu: '100m', memory: '64Mi' },
                            },
                        },
                    },
                },
            },
        },
    },

    metrics_service: kube.Service(this.name_prefix + '-metrics') {
        target_pod: this.manager_deployment.spec.template,
        metadata+: {
            namespace: this.namespace,
            labels: this.labels,
        },
        spec+: { type: 'ClusterIP' },
    },
}