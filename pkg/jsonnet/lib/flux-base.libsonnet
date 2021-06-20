local utils = import 'utils.libsonnet';

{
    // Object is the base of all FluxCD objects. The returned
    // object has methods that can be used to inherit fields and
    // assertions common across objects.
    Object(group, kind, name, version='v1beta1'):: {
        local object = self,

        apiVersion: '%s.toolkit.fluxcd.io/%s' % [group, version],
        kind: kind,
        metadata: {
            name: name
        },

        // WithIntervalAndTimeout returns this object with an assumed required interval
        // and an optional timeout config.
        WithIntervalAndTimeout(interval):: object {
            local objectWithTimeout = self,
            config+:: {
                timeout: null,
            },
            spec_+:: {
                interval: interval,
                timeout: if std.objectHas(objectWithTimeout.config, 'timeout') 
                    then objectWithTimeout.config.timeout,
                assert utils.nullOrIsType(self.timeout, 'string') : '"timeout" must be a string',
            },
        },

        // WithCredentials returns this object with a config for a secret pointing
        // at credentials for the resource.
        WithCredentials():: object {
            local objectWithCredentials = self,
            config+:: {
                credentialsSecret: '',
            },
            spec_+:: {
                secretRef: if std.objectHas(objectWithCredentials.config, 'credentialsSecret') && objectWithCredentials.config.credentialsSecret != '' 
                    then {
                        name: objectWithCredentials.config.credentialsSecret,
                        assert utils.nullOrIsType(self.name, 'string') : 
                            '"credentialsSecret" must be a string'
                    }
            },
        },

        // WithIgnore returns this object with configurations for a generic string "ignore"
        // parameter.
        WithIgnore():: object {
            local objectWithIgnore = self,
            config+:: {
                ignore: ''
            },
            spec_+:: {
                ignore: if std.objectHas(objectWithIgnore.config, 'ignore') && objectWithIgnore.config.ignore != '' 
                    then objectWithIgnore.config.ignore,
                    assert utils.nullOrIsType(self.ignore, 'boolean') :
                        '"ignore" must be a string'
            },
        },

        // WithSuspend returns this object with a configuration to suspend the reconciliation
        // of the CR.
        WithSuspend():: object {
            local objectWithSuspend = self,
            config+:: {
                suspend: false,
            },
            spec_+:: {
                assert utils.notExistsOrType(objectWithSuspend.config, 'suspend', 'boolean') :
                    '"suspend" must be a boolean value',
                suspend: if std.objectHas(objectWithSuspend.config, 'suspend') && objectWithSuspend.config.suspend 
                    then objectWithSuspend.config.suspend,                    
            },
        },

        // WithLocalSourceRef return this object with a configuration for a source reference.
        // The reference can be actual objects or a dictionary with at the very least 'kind'
        // and 'name' fields.
        WithLocalSourceRef():: object {
            local withSourceRef = self,

            sourceRef:: error 'must provide a source reference with sourceRef::',
            local ref = withSourceRef.sourceRef,

            assert std.type(ref) == 'object' :
                'sourceRef_ must be an object',

            assert std.objectHas(ref, 'kind') && (
                        std.objectHas(ref, 'name') || (
                            std.objectHas(ref, 'metadata') && std.objectHas(ref.metadata, 'name')
                        )
                   ):
                'sourceRef_ must have be a reference to an object or have a "name" and "kind"',

            getName()::
                if std.objectHas(ref, 'name') then ref.name
                else if std.objectHas(ref, 'metadata') && std.objectHas(ref.metadata, 'name') then ref.metadata.name,

            spec_+:: {
                sourceRef: {                    
                    apiVersion: if std.objectHas(ref, 'apiVersion') then ref.apiVersion else null,
                    assert utils.nullOrIsType(self.apiVersion, 'string'),

                    kind: ref.kind,
                    name: withSourceRef.getName()
                },
            }
        },

        // WithNameFromPrivate will look for a private 'name' field
        // in the object and use that as the name.
        WithNameFromPrivate():: object {
            local obj = self,
            metadata+: {
                name: obj.name,
            },
        },

        // PruneFromPrivateSpec prunes the private spec_ of null or false values and sets
        // it to the object's spec field.
        PruneFromPrivateSpec():: object {
            local obj = self,
            spec+: std.prune(obj.spec_)
        },
    },
}