local flux = import 'flux-base.libsonnet';
local utils = import 'utils.libsonnet';
local default_interval = '5m';

{
    // Creates a new GitRepository. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/gitrepositories
    GitRepository(url, interval=default_interval):: 
        flux.Object('source', 'GitRepository', utils.urlToDefaultName(url))
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithIgnore()
        .WithSuspend()
    {
        local repo = self,

        name:: null,

        config+:: {
            verifySecret: null,
            ref: {},
            recurseSubmodules: false,
            includes: {}
        },

        mapIncludes(includes):: [
            {
                assert std.type(includes[key]) == 'object' : 
                    'GitRepository %s "includes" must be map of repository names to path configurations (or an empty object for the defaults)' % repo.GetName(),
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
                    'GitRepository %s ref configuration %s is invalid' % [ repo.GetName(), std.toString(config.ref)],
                branch: if std.objectHas(config.ref, 'branch') then config.ref.branch else null,
                tag: if std.objectHas(config.ref, 'tag') then config.ref.tag else null,
                semver: if std.objectHas(config.ref, 'semver') then config.ref.semver else null,
                commit: if std.objectHas(config.ref, 'commit') then config.ref.commit else null,
            },

            verify: if std.objectHas(config, 'verifySecret') && config.verifySecret != null then {
                assert std.type(config.verifySecret) == 'string' : 
                    'GitRepository %s "verifySecret" must be a string' % repo.GetName(),
                mode: 'head',
                secretRef: { name: config.verifySecret },
            },

            recurseSubmodules: if std.objectHas(config, 'recurseSubmodules') && config.recurseSubmodules then true,
            assert utils.notExistsOrType(config, 'recurseSubmodules', 'boolean') :
                'GitRepository %s "recurseSubmodules" must be a boolean value' % repo.GetName(),

            include: if std.objectHas(config, 'includes') && std.type(config.includes) == 'object' then repo.mapIncludes(config.includes),
            assert utils.notExistsOrType(config, 'includes', 'object') :
                'GitRepository %s "includes" must be map of repository names to path configurations (or an empty object for the defaults)' % repo.GetName(),
        },
    },
    
    // Creates a new Bucket. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/buckets
    Bucket(bucketName, endpoint="s3.amazonaws.com", interval=default_interval):: 
        flux.Object('source', 'Bucket', bucketName)
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithIgnore()
        .WithSuspend()
    {
        local bucket = self,

        name:: null,

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
                'Bucket %s "insecure" must be a boolean value' % bucket.GetName(),

            region: if std.objectHas(config, 'region') && config.region != '' then config.region,
            assert utils.nullOrIsType(self.region, 'string') :
                'Bucket %s "region" must be a string value' % bucket.GetName()
        },
    },

    // Creates a new HelmRepository. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/helmrepositories
    HelmRepository(url, interval=default_interval):: 
        flux.Object('source', 'HelmRepository', utils.urlToDefaultName(url))
        .WithIntervalAndTimeout(interval)
        .WithCredentials()
        .WithSuspend()
    {
        local helmrepo = self,

        name:: null,

        config+:: {
            passCredentials: false
        },

        spec_+:: {
            local config = helmrepo.config,
            url: url,
            passCredentials: if std.objectHas(config, 'passCredentials') && config.passCredentials != false then config.passCredentials,
            assert utils.nullOrIsType(self.passCredentials, 'boolean') :
                'HelmRepository %s "passCredentials" must be a boolean value' % helmrepo.GetName(),
        },
    },

    // Creates a new HelmChart. See the "config" field in this object
    // and its parents for additional configuration options.
    // https://fluxcd.io/docs/components/source/helmcharts
    HelmChart(chart, interval=default_interval):: 
        flux.Object('source', 'HelmChart', chart)
        .WithIntervalAndTimeout(interval)
        .WithSuspend()
        .WithLocalSourceRef()
    {
        local helmchart = self,

        name:: null,

        config+:: {
            version: '',
            valuesFiles: []
        },

        spec_+:: {
            local config = helmchart.config,

            chart: chart,
            
            version: if std.objectHas(config, 'version') && config.version != '' then config.version,
            assert utils.nullOrIsType(self.version, 'string') :
                'HelmChart %s "version" must be a string' % helmchart.GetName(),

            valuesFiles: if std.objectHas(config, 'valuesFiles') && config.valuesFiles != [] then config.valuesFiles,
            assert utils.nullOrIsType(self.valuesFiles, 'array') :
                'HelmChart %s "valuesFiles" must be an array of strings' % helmchart.GetName(),
        },
    }
}