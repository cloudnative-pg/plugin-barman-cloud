package metadata

// MetadataNamespace is the namespace used for the Barman Cloud plugin metadata
const MetadataNamespace = "barmancloud.cnpg.io"

const (
	// UseDefaultAzureCredentialsAnnotationName is an annotation that can be set
	// on an ObjectStore resource to enable the authentication to Azure via DefaultAzureCredentials.
	// This is meant to be used with inheritFromAzureAD enabled.
	UseDefaultAzureCredentialsAnnotationName = MetadataNamespace + "/useDefaultAzureCredentials"

	// UseDefaultAzureCredentialsTrueValue is the value for the annotation
	// barmancloud.cnpg.io/useDefaultAzureCredentials to enable the DefaultAzureCredentials auth mechanism.
	UseDefaultAzureCredentialsTrueValue = "true"
)
