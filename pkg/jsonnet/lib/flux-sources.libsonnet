local flux = import 'flux-base.libsonnet';
local utils = import 'utils.libsonnet';
local default_interval = '5m';

{
    // Creates a new GitRepository. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/gitrepositories
    GitRepository(url, interval=default_interval):: 
        flux.Object('source', 'GitRepository', '')
        .WithNameFromPrivate()
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithIgnore()
        .WithSuspend()
        .PruneFromPrivateSpec() 
    {
        local repo = self,

        // A name generated from the repository URL. Override this private field to set explicitly.
        name:: utils.urlToDefaultName(url),

        config+:: {
            verifySecret: null,
            ref: {},
            recurseSubmodules: false,
            includes: {}
        },

        map_includes(includes):: [
            {
                assert std.type(includes[key]) == 'object' : '"config_.includes" must be map of repository names to path configurations (or an empty object for the defaults)',
                repository: { name: key },
                fromPath: if std.objectHas(includes[key], 'fromPath') then includes[key].fromPath,
                toPath: if std.objectHas(includes[key], 'toPath') then includes[key].toPath,
            }
            for key in std.objectFields(includes)
        ],

        spec_+:: {
            local config = repo.config,

            url: url,

            ref: if std.objectHas(config, 'ref') && config.ref != {} then {
                assert std.type(config.ref) == 'object' && 
                    (std.objectHas(config.ref, 'branch') ||
                        std.objectHas(config.ref, 'tag') ||
                        std.objectHas(config.ref, 'semver') ||
                        std.objectHas(config.ref, 'commit')) :
                    'ref configuration %s is invalid' % std.toString(config.ref),
                branch: if std.objectHas(config.ref, 'branch') then config.ref.branch else null,
                tag: if std.objectHas(config.ref, 'tag') then config.ref.tag else null,
                semver: if std.objectHas(config.ref, 'semver') then config.ref.semver else null,
                commit: if std.objectHas(config.ref, 'commit') then config.ref.commit else null,
            },

            verify: if std.objectHas(config, 'verifySecret') && config.verifySecret != null then {
                assert std.type(config.verifySecret) == 'string' : '"verifySecret" must be a string',
                mode: 'head',
                secretRef: { name: config.verifySecret },
            },

            recurseSubmodules: if std.objectHas(config, 'recurseSubmodules') && config.recurseSubmodules then true,
            assert utils.notExistsOrType(config, 'recurseSubmodules', 'boolean') :
                '"recurseSubmodules" must be a boolean value',

            include: if std.objectHas(config, 'includes') && std.type(config.includes) == 'object' then repo.map_includes(config.includes),
            assert utils.notExistsOrType(config, 'includes', 'object') :
                '"includes" must be map of repository names to path configurations (or an empty object for the defaults)',
        },
    },
    
    // Creates a new Bucket. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/buckets
    Bucket(bucketName, endpoint="s3.amazonaws.com", interval=default_interval):: 
        flux.Object('source', 'Bucket', '')
        .WithNameFromPrivate()
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithIgnore()
        .WithSuspend()
        .PruneFromPrivateSpec() 
    {
        local bucket = self,

        name:: bucketName,

        config+:: {
            insecure: false,
            region: ''
        },

        spec_+:: {
            local config = bucket.config,

            bucketName: bucketName,
            endpoint: endpoint,

            insecure: if std.objectHas(config, 'insecure') && config.insecure != false then config.insecure,
            assert utils.nullOrIsType(self.insecure, 'boolean') :
                '"insecure" must be a boolean value',

            region: if std.objectHas(config, 'region') && config.region != '' then config.region,
            assert utils.nullOrIsType(self.region, 'string') :
                '"region" must be a string value'
        },
    },

    // Creates a new HelmRepository. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/helmrepositories
    HelmRepository(url, interval=default_interval):: 
        flux.Object('source', 'HelmRepository', '')
        .WithNameFromPrivate()
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithSuspend()
        .PruneFromPrivateSpec() 
    {
        local helmrepo = self,

        name:: utils.urlToDefaultName(url),

        config+:: {
            passCredentials: false
        },

        spec_+:: {
            local config = helmrepo.config,
            url: url,
            passCredentials: if std.objectHas(config, 'passCredentials') && config.passCredentials != false then config.passCredentials,
            assert utils.nullOrIsType(self.passCredentials, 'boolean') :
                '"passCredentials" must be a boolean value',
        },
    },

    // Creates a new HelmChart. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/helmcharts
    HelmChart(chart, interval=default_interval):: 
        flux.Object('source', 'HelmChart', '')
        .WithNameFromPrivate()
        .WithIntervalAndTimeout(interval)
        .WithSuspend()
        .PruneFromPrivateSpec() 
        .WithLocalSourceRef()
    {
        local helmchart = self,

        name:: chart,
    }
}