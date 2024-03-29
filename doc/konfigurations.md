Packages:

-   [jsonnet.io/v1beta1](#jsonnet.io%2fv1beta1)

## jsonnet.io/v1beta1

Package v1beta1 file doc.go required for the doc generator to register
this as an API

Resource Types:

### Konfiguration

Konfiguration is the Schema for the konfigurations API

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>metadata</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">Kubernetes meta/v1.ObjectMeta</a></em></td>
<td>Refer to the Kubernetes API documentation for the fields of the <code>metadata</code> field.</td>
</tr>
<tr class="even">
<td><code>spec</code><br />
<em><a href="#KonfigurationSpec">KonfigurationSpec</a></em></td>
<td><br />
<br />

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<tbody>
<tr class="odd">
<td><code>dependsOn</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">[]FluxCD runtime/dependency.CrossNamespaceDependencyReference</a></em></td>
<td><em>(Optional)</em>
<p>DependsOn may contain a dependency.CrossNamespaceDependencyReference slice with references to Konfiguration resources that must be ready before this Konfiguration can be reconciled.</p></td>
</tr>
<tr class="even">
<td><code>interval</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><p>The interval at which to reconcile the Konfiguration.</p></td>
</tr>
<tr class="odd">
<td><code>retryInterval</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>The interval at which to retry a previously failed reconciliation. When not specified, the controller uses the KonfigurationSpec.Interval value to retry failures.</p></td>
</tr>
<tr class="even">
<td><code>kubeConfig</code><br />
<em><a href="#KubeConfig">KubeConfig</a></em></td>
<td><em>(Optional)</em>
<p>The KubeConfig for reconciling the Konfiguration on a remote cluster. Defaults to the in-cluster configuration.</p></td>
</tr>
<tr class="odd">
<td><code>path</code><br />
<em>string</em></td>
<td><p>Path to the jsonnet, json, or yaml that should be applied to the cluster. Defaults to ‘None’, which translates to the root path of the SourceRef. When declared as a file path it is assumed to be from the root path of the SourceRef. You may also define a HTTP(S) link to fetch files from a remote location.</p></td>
</tr>
<tr class="even">
<td><code>jsonnetPaths</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional search paths to add to the jsonnet importer. These are relative to the root of the sourceRef.</p></td>
</tr>
<tr class="odd">
<td><code>jsonnetURLs</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional HTTP(S) URLs to add to the jsonnet importer.</p></td>
</tr>
<tr class="even">
<td><code>variables</code><br />
<em><a href="#Variables">Variables</a></em></td>
<td><em>(Optional)</em>
<p>External variables and top-level arguments to supply to the jsonnet at <code>path</code>.</p></td>
</tr>
<tr class="odd">
<td><code>inject</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>Inject raw jsonnet into the evaluation.</p></td>
</tr>
<tr class="even">
<td><code>serviceAccountName</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>The name of the Kubernetes service account to impersonate when reconciling this Konfiguration.</p></td>
</tr>
<tr class="odd">
<td><code>sourceRef</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">FluxCD apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>Reference of the source where the jsonnet, json, or yaml file(s) are.</p></td>
</tr>
<tr class="even">
<td><code>prune</code><br />
<em>bool</em></td>
<td><p>Prune enables garbage collection. This means that when newly rendered jsonnet does not contain objects that were applied previously, they will be removed. When a Konfiguration is removed that had this value set to <code>true</code>, all resources created by it will also be removed.</p></td>
</tr>
<tr class="odd">
<td><code>healthChecks</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">[]FluxCD apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>A list of resources to be included in the health assessment.</p></td>
</tr>
<tr class="even">
<td><code>suspend</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent reconciliations, it does not apply to already started executions. Defaults to false.</p></td>
</tr>
<tr class="odd">
<td><code>timeout</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>Timeout for diff, validation, apply, and health checking operations. Defaults to ‘Interval’ duration.</p></td>
</tr>
<tr class="even">
<td><code>validate</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Validate input against the server schema, defaults to true. At the moment this just implies a dry-run before patch/create operations. This will be updated to support different methods of validation.</p></td>
</tr>
<tr class="odd">
<td><code>force</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Force instructs the controller to recreate resources when patching fails due to an immutable field change.</p></td>
</tr>
</tbody>
</table></td>
</tr>
<tr class="odd">
<td><code>status</code><br />
<em><a href="#KonfigurationStatus">KonfigurationStatus</a></em></td>
<td></td>
</tr>
</tbody>
</table>

### KonfigurationSpec

(*Appears on:*[Konfiguration](#Konfiguration))

KonfigurationSpec defines the desired state of a Konfiguration

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>dependsOn</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">[]FluxCD runtime/dependency.CrossNamespaceDependencyReference</a></em></td>
<td><em>(Optional)</em>
<p>DependsOn may contain a dependency.CrossNamespaceDependencyReference slice with references to Konfiguration resources that must be ready before this Konfiguration can be reconciled.</p></td>
</tr>
<tr class="even">
<td><code>interval</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><p>The interval at which to reconcile the Konfiguration.</p></td>
</tr>
<tr class="odd">
<td><code>retryInterval</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>The interval at which to retry a previously failed reconciliation. When not specified, the controller uses the KonfigurationSpec.Interval value to retry failures.</p></td>
</tr>
<tr class="even">
<td><code>kubeConfig</code><br />
<em><a href="#KubeConfig">KubeConfig</a></em></td>
<td><em>(Optional)</em>
<p>The KubeConfig for reconciling the Konfiguration on a remote cluster. Defaults to the in-cluster configuration.</p></td>
</tr>
<tr class="odd">
<td><code>path</code><br />
<em>string</em></td>
<td><p>Path to the jsonnet, json, or yaml that should be applied to the cluster. Defaults to ‘None’, which translates to the root path of the SourceRef. When declared as a file path it is assumed to be from the root path of the SourceRef. You may also define a HTTP(S) link to fetch files from a remote location.</p></td>
</tr>
<tr class="even">
<td><code>jsonnetPaths</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional search paths to add to the jsonnet importer. These are relative to the root of the sourceRef.</p></td>
</tr>
<tr class="odd">
<td><code>jsonnetURLs</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional HTTP(S) URLs to add to the jsonnet importer.</p></td>
</tr>
<tr class="even">
<td><code>variables</code><br />
<em><a href="#Variables">Variables</a></em></td>
<td><em>(Optional)</em>
<p>External variables and top-level arguments to supply to the jsonnet at <code>path</code>.</p></td>
</tr>
<tr class="odd">
<td><code>inject</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>Inject raw jsonnet into the evaluation.</p></td>
</tr>
<tr class="even">
<td><code>serviceAccountName</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>The name of the Kubernetes service account to impersonate when reconciling this Konfiguration.</p></td>
</tr>
<tr class="odd">
<td><code>sourceRef</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">FluxCD apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>Reference of the source where the jsonnet, json, or yaml file(s) are.</p></td>
</tr>
<tr class="even">
<td><code>prune</code><br />
<em>bool</em></td>
<td><p>Prune enables garbage collection. This means that when newly rendered jsonnet does not contain objects that were applied previously, they will be removed. When a Konfiguration is removed that had this value set to <code>true</code>, all resources created by it will also be removed.</p></td>
</tr>
<tr class="odd">
<td><code>healthChecks</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">[]FluxCD apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>A list of resources to be included in the health assessment.</p></td>
</tr>
<tr class="even">
<td><code>suspend</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent reconciliations, it does not apply to already started executions. Defaults to false.</p></td>
</tr>
<tr class="odd">
<td><code>timeout</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>Timeout for diff, validation, apply, and health checking operations. Defaults to ‘Interval’ duration.</p></td>
</tr>
<tr class="even">
<td><code>validate</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Validate input against the server schema, defaults to true. At the moment this just implies a dry-run before patch/create operations. This will be updated to support different methods of validation.</p></td>
</tr>
<tr class="odd">
<td><code>force</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Force instructs the controller to recreate resources when patching fails due to an immutable field change.</p></td>
</tr>
</tbody>
</table>

### KonfigurationStatus

(*Appears on:*[Konfiguration](#Konfiguration))

KonfigurationStatus defines the observed state of Konfiguration

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>observedGeneration</code><br />
<em>int64</em></td>
<td><em>(Optional)</em>
<p>ObservedGeneration is the last reconciled generation.</p></td>
</tr>
<tr class="even">
<td><code>conditions</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#condition-v1-meta">[]Kubernetes meta/v1.Condition</a></em></td>
<td><em>(Optional)</em></td>
</tr>
<tr class="odd">
<td><code>lastAppliedRevision</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>The last successfully applied revision. The revision format for Git sources is /. For HTTP(S) paths it will just be the URL.</p></td>
</tr>
<tr class="even">
<td><code>lastAttemptedRevision</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>LastAttemptedRevision is the revision of the last reconciliation attempt. For HTTP(S) paths it will just be the URL.</p></td>
</tr>
<tr class="odd">
<td><code>snapshot</code><br />
<em><a href="#Snapshot">Snapshot</a></em></td>
<td><em>(Optional)</em>
<p>The last successfully applied revision metadata.</p></td>
</tr>
</tbody>
</table>

### KubeConfig

(*Appears
on:*[KonfigurationSpec](#KonfigurationSpec))

KubeConfig holds the configuration for where to fetch the contents of a
kubeconfig file.

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>secretRef</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#localobjectreference-v1-core">Kubernetes core/v1.LocalObjectReference</a></em></td>
<td><p>SecretRef holds the name to a secret that contains a ‘value’ key with the kubeconfig file as the value. It must be in the same namespace as the Konfiguration. It is recommended that the kubeconfig is self-contained, and the secret is regularly updated if credentials such as a cloud-access-token expire. Cloud specific <code>cmd-path</code> auth helpers will not function without adding binaries and credentials to the Pod that is responsible for reconciling the Konfiguration.</p></td>
</tr>
</tbody>
</table>

### Snapshot

(*Appears
on:*[KonfigurationStatus](#KonfigurationStatus))

Snapshot holds the metadata of the Kubernetes objects generated for a
source revision

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>checksum</code><br />
<em>string</em></td>
<td><p>The manifests sha1 checksum.</p></td>
</tr>
<tr class="even">
<td><code>entries</code><br />
<em><a href="#SnapshotEntry">[]SnapshotEntry</a></em></td>
<td><p>A list of Kubernetes kinds grouped by namespace.</p></td>
</tr>
</tbody>
</table>

### SnapshotEntry

(*Appears on:*[Snapshot](#Snapshot))

SnapshotEntry holds the metadata of namespaced Kubernetes objects

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>namespace</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>The namespace of this entry.</p></td>
</tr>
<tr class="even">
<td><code>kinds</code><br />
<em>map[string]string</em></td>
<td><p>The list of Kubernetes kinds.</p></td>
</tr>
</tbody>
</table>

### StatusMeta

StatusMeta is a helper struct for setting the status on custom
resources.

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>Revision</code><br />
<em>string</em></td>
<td></td>
</tr>
<tr class="even">
<td><code>Reason</code><br />
<em>string</em></td>
<td></td>
</tr>
<tr class="odd">
<td><code>Message</code><br />
<em>string</em></td>
<td></td>
</tr>
</tbody>
</table>

### Variables

(*Appears
on:*[KonfigurationSpec](#KonfigurationSpec))

Variables describe code/strings for external variables and top-level
arguments.

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>extStr</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of external variables with string values.</p></td>
</tr>
<tr class="even">
<td><code>extCode</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of external variables with values supplied as Jsonnet code.</p></td>
</tr>
<tr class="odd">
<td><code>extVars</code><br />
<em>k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON</em></td>
<td><em>(Optional)</em>
<p>Values for external variables. They will be used as strings or code depending on the types encountered.</p></td>
</tr>
<tr class="even">
<td><code>tlaStr</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of top-level-arguments with string values.</p></td>
</tr>
<tr class="odd">
<td><code>tlaCode</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of top-level-arguments with values supplied as Jsonnet code.</p></td>
</tr>
<tr class="even">
<td><code>tlaVars</code><br />
<em>k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON</em></td>
<td><em>(Optional)</em>
<p>Values for top level arguments. They will be used as strings or code depending on the types encountered.</p></td>
</tr>
</tbody>
</table>

------------------------------------------------------------------------

*Generated with `gen-crd-api-reference-docs` on git commit `03dee2e`.*
