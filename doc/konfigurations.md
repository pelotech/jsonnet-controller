Packages:

-   [apps.kubecfg.io/v1](#apps.kubecfg.io%2fv1)

## apps.kubecfg.io/v1

Package v1 file doc.go required for the doc generator to register this
as an API

Resource Types:

### CrossNamespaceSourceReference

(*Appears
on:*[KonfigurationSpec](#KonfigurationSpec))

CrossNamespaceSourceReference contains enough information to let you
locate the typed referenced object at cluster level

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
<td><code>apiVersion</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>API version of the referent</p></td>
</tr>
<tr class="even">
<td><code>kind</code><br />
<em>string</em></td>
<td><p>Kind of the referent</p></td>
</tr>
<tr class="odd">
<td><code>name</code><br />
<em>string</em></td>
<td><p>Name of the referent</p></td>
</tr>
<tr class="even">
<td><code>namespace</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>Namespace of the referent, defaults to the Konfiguration namespace</p></td>
</tr>
</tbody>
</table>

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
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">[]github.com/fluxcd/pkg/runtime/dependency.CrossNamespaceDependencyReference</a></em></td>
<td><em>(Optional)</em>
<p>DependsOn may contain a dependency.CrossNamespaceDependencyReference slice with references to Konfiguration resources that must be ready before this Konfiguration can be reconciled. NOTE: Not yet implemented.</p></td>
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
<td><code>variables</code><br />
<em><a href="#Variables">Variables</a></em></td>
<td><em>(Optional)</em>
<p>Variables to use when invoking kubecfg to render manifests.</p></td>
</tr>
<tr class="odd">
<td><code>sourceRef</code><br />
<em><a href="#CrossNamespaceSourceReference">CrossNamespaceSourceReference</a></em></td>
<td><em>(Optional)</em>
<p>Reference of the source where the jsonnet, json, or yaml file(s) are.</p></td>
</tr>
<tr class="even">
<td><code>prune</code><br />
<em>bool</em></td>
<td><p>Prune enables garbage collection. Note that this makes commands take considerably longer, so you may want to adjust your timeouts accordingly.</p></td>
</tr>
<tr class="odd">
<td><code>healthChecks</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">[]github.com/fluxcd/pkg/apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>A list of resources to be included in the health assessment. NOTE: Not yet implemented.</p></td>
</tr>
<tr class="even">
<td><code>suspend</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent kubecfg executions, it does not apply to already started executions. Defaults to false.</p></td>
</tr>
<tr class="odd">
<td><code>timeout</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>Timeout for diff, validation, apply, and (soon) health checking operations. Defaults to ‘Interval’ duration.</p></td>
</tr>
<tr class="even">
<td><code>kubecfgArgs</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional global arguments to pass to kubecfg invocations.</p></td>
</tr>
<tr class="odd">
<td><code>validate</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Validate input against the server schema, defaults to true. This will be updated to support different methods of validation.</p></td>
</tr>
<tr class="even">
<td><code>diffStrategy</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>Strategy to use when performing diffs against the current state of the cluster. Options are <code>all</code>, <code>subset</code>, or <code>last-applied</code>. Defaults to <code>subset</code>.</p></td>
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

KonfigurationSpec defines the desired state of Konfiguration

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
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">[]github.com/fluxcd/pkg/runtime/dependency.CrossNamespaceDependencyReference</a></em></td>
<td><em>(Optional)</em>
<p>DependsOn may contain a dependency.CrossNamespaceDependencyReference slice with references to Konfiguration resources that must be ready before this Konfiguration can be reconciled. NOTE: Not yet implemented.</p></td>
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
<td><code>variables</code><br />
<em><a href="#Variables">Variables</a></em></td>
<td><em>(Optional)</em>
<p>Variables to use when invoking kubecfg to render manifests.</p></td>
</tr>
<tr class="odd">
<td><code>sourceRef</code><br />
<em><a href="#CrossNamespaceSourceReference">CrossNamespaceSourceReference</a></em></td>
<td><em>(Optional)</em>
<p>Reference of the source where the jsonnet, json, or yaml file(s) are.</p></td>
</tr>
<tr class="even">
<td><code>prune</code><br />
<em>bool</em></td>
<td><p>Prune enables garbage collection. Note that this makes commands take considerably longer, so you may want to adjust your timeouts accordingly.</p></td>
</tr>
<tr class="odd">
<td><code>healthChecks</code><br />
<em><a href="https://pkg.go.dev/github.com/fluxcd/pkg/apis/meta#NamespacedObjectKindReference">[]github.com/fluxcd/pkg/apis/meta.NamespacedObjectKindReference</a></em></td>
<td><em>(Optional)</em>
<p>A list of resources to be included in the health assessment. NOTE: Not yet implemented.</p></td>
</tr>
<tr class="even">
<td><code>suspend</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent kubecfg executions, it does not apply to already started executions. Defaults to false.</p></td>
</tr>
<tr class="odd">
<td><code>timeout</code><br />
<em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#duration-v1-meta">Kubernetes meta/v1.Duration</a></em></td>
<td><em>(Optional)</em>
<p>Timeout for diff, validation, apply, and (soon) health checking operations. Defaults to ‘Interval’ duration.</p></td>
</tr>
<tr class="even">
<td><code>kubecfgArgs</code><br />
<em>[]string</em></td>
<td><em>(Optional)</em>
<p>Additional global arguments to pass to kubecfg invocations.</p></td>
</tr>
<tr class="odd">
<td><code>validate</code><br />
<em>bool</em></td>
<td><em>(Optional)</em>
<p>Validate input against the server schema, defaults to true. This will be updated to support different methods of validation.</p></td>
</tr>
<tr class="even">
<td><code>diffStrategy</code><br />
<em>string</em></td>
<td><em>(Optional)</em>
<p>Strategy to use when performing diffs against the current state of the cluster. Options are <code>all</code>, <code>subset</code>, or <code>last-applied</code>. Defaults to <code>subset</code>.</p></td>
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

Snapshot holds the metadata of namespaced Kubernetes objects

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
<td><code>tlaStr</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of top level arguments with string values.</p></td>
</tr>
<tr class="even">
<td><code>tlaCode</code><br />
<em>map[string]string</em></td>
<td><em>(Optional)</em>
<p>Values of top level arguments with values supplied as Jsonnet code.</p></td>
</tr>
</tbody>
</table>

------------------------------------------------------------------------

*Generated with `gen-crd-api-reference-docs` on git commit `1cf196c`.*
