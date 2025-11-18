package metadata

// MetadataNamespace is the namespace used for the Barman Cloud plugin metadata
const MetadataNamespace = "barmancloud.cnpg.io"

const (
	// UseDefaultAzureCredentialAnnotationName is an annotation that can be set
	// on an ObjectStore resource to enable the authentication to Azure via DefaultAzureCredential.
	// This is meant to be used with inheritFromAzureAD enabled.
	UseDefaultAzureCredentialAnnotationName = MetadataNamespace + "/useDefaultAzureCredential"

	// UseDefaultAzureCredentialTrueValue is the value for the annotation
	// barmancloud.cnpg.io/useDefaultAzureCredential to enable the DefaultAzureCredentials auth mechanism.
	UseDefaultAzureCredentialTrueValue = "true"
)
