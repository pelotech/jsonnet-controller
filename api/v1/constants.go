package v1

const (
	// GitRepositoryIndexKey is the key used for indexing kustomizations
	// based on their Git sources.
	GitRepositoryIndexKey string = ".metadata.gitRepository"
	// BucketIndexKey is the key used for indexing kustomizations
	// based on their S3 sources.
	BucketIndexKey string = ".metadata.bucket"
)
