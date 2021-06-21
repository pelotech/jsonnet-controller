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
            // This will usually get evaluated last, so errors will 
            // contain the "assumed" name. But if the user provides
            // any "name:" field it will override this value in the
            // final rendering.
            //
            // Extending objects should explicity define a `name:: null`
            // to avoid the user adding garbage fields to the rendering.
            name: if std.objectHas(object, 'name') && object.name != null then object.name else name
        },

        // All extending objects should extend the spec_:: private field
        spec+: std.prune(object.spec_),

        // Helpers for retrieving the default name and kind of this object
        GetName():: object.metadata.name,
        GetKind():: object.kind,

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
                assert utils.nullOrIsType(self.timeout, 'string') : 
                    '%s %s "timeout" must be a string' % [object.GetKind(), object.GetName()],
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
                            '%s %s "credentialsSecret" must be a string' % [object.GetKind(), object.GetName()]
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
                        '%s %s "ignore" must be a string' % [object.GetKind(), object.GetName()]
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
                    '%s %s "suspend" must be a boolean value' % [object.GetKind(), object.GetName()],
                suspend: if std.objectHas(objectWithSuspend.config, 'suspend') && objectWithSuspend.config.suspend 
                    then objectWithSuspend.config.suspend,                    
            },
        },

        // WithLocalSourceRef return this object with a configuration for a source reference.
        // The reference can be actual objects or a dictionary with at the very least 'kind'
        // and 'name' fields.
        WithLocalSourceRef():: object {
            local withSourceRef = self,

            sourceRef:: error 'must provide a source reference for %s %s with sourceRef::' % [object.GetKind(), object.GetName()],
            local ref = withSourceRef.sourceRef,

            assert std.type(ref) == 'object' :
                '%s %s sourceRef_ must be an object' % [object.GetKind(), object.GetName()],

            assert std.objectHas(ref, 'kind') && (
                        std.objectHas(ref, 'name') || (
                            std.objectHas(ref, 'metadata') && std.objectHas(ref.metadata, 'name')
                        )
                   ):
                '%s %s sourceRef_ must have be a reference to an object or have a "name" and "kind"' % [object.GetKind(), object.GetName()],

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
    },
}