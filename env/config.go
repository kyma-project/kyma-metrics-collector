package env

// Config contains the configurations which are controlled by the ENV vars.
type Config struct {
	PublicCloudSpecsPath string `envconfig:"PUBLIC_CLOUD_SPECS" required:"true"`
}
